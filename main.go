package main

import (
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	COMMAND_UNKNOWN CmdType = 0x00
	COMMAND_INLINE  CmdType = 0x01
	COMMAND_BULK    CmdType = 0x02
)

const (
	GODIS_IO_BUF     int = 1024 * 12
	GODIS_MAX_BULK   int = 1024 * 4
	GODIS_MAX_INLINE int = 1024 * 4
)

type CmdType = byte

type GodisDB struct {
	data   *Dict
	expire *Dict
}

type GodisServer struct {
	fd      int
	port    int
	db      *GodisDB
	clients map[int]*GodisClient
	keLoop  *KeLoop
}

type GodisClient struct {
	fd       int
	db       *GodisDB
	args     []*Gobj
	reply    *List
	sentLen  int
	queryBuf []byte
	queryLen int
	cmdType  CmdType
	bulkNum  int
	bulkLen  int
}

type CommandProc func(c *GodisClient)

type GodisCommand struct {
	name  string
	proc  CommandProc
	arity int
}

var server GodisServer

var cmdTable = []GodisCommand{
	{"get", getCommand, 2},
	{"set", setCommand, 3},
	{"expire", expireCommand, 3},
}

func expireIfNeeded(key *Gobj) {
	entry := server.db.expire.Find(key)
	if entry == nil {
		return
	}
	when := entry.Val.IntVal()
	if when > GetMsTime() { // 不到过期时间
		return
	}
	server.db.expire.Delete(key)
	server.db.data.Delete(key)
}

func findKeyRead(key *Gobj) *Gobj {
	expireIfNeeded(key) // 检查key要不要过期
	return server.db.data.Get(key)
}

func getCommand(c *GodisClient) {
	key := c.args[1]
	val := findKeyRead(key)
	if val == nil {
		c.AddReplyStr("$-1\r\n")
	} else if val.Type != GSTR {
		c.AddReplyStr("-ERR: worng type\r\n")
	} else {
		strVal := val.StrVal()
		c.AddReplyStr(fmt.Sprintf("$%d%v\r\n", len(strVal), strVal))
	}
}

func setCommand(c *GodisClient) {
	key := c.args[1]
	val := c.args[2]
	if val.Type != GSTR {
		c.AddReplyStr("-ERR: wrong type\r\n")
	}
	server.db.data.Set(key, val)
	server.db.expire.Delete(key)
	c.AddReplyStr("+OK\r\n")
}

func expireCommand(c *GodisClient) {
	key := c.args[1]
	val := c.args[2]
	if val.Type != GSTR {
		c.AddReplyStr("-ERR: wrong type\r\n")
	}
	expire := GetMsTime() + (val.IntVal() * 1000) // 转成毫秒
	expireObj := CreateFromInt(expire)
	server.db.expire.Set(key, expireObj)
	expireObj.DecrRefCount()
	c.AddReplyStr("+OK\r\n")
}

/*
查找命令
*/
func lookupCommand(cmdStr string) *GodisCommand {
	for _, c := range cmdTable {
		if c.name == cmdStr {
			return &c
		}
	}
	return nil
}

func (c *GodisClient) AddReply(o *Gobj) {
	c.reply.Append(o)
	o.IncrRefCount()
	server.keLoop.AddFileEvent(c.fd, KE_WRITABLE, SendReplyToClient, c) // 将reply注册为一个写事件。
}

func (c *GodisClient) AddReplyStr(str string) {
	object := CreateObject(GSTR, str)
	c.AddReply(object)
	object.DecrRefCount() // 这里要减下去，因为AddReply会加一
}

/*
先拿到命令是啥
*/
func ProcessCommand(c *GodisClient) {
	cmdStr := c.args[0].StrVal()
	log.Printf("process command: %v\n", cmdStr)
	if cmdStr == "quit" {
		freeClient(c)
		return
	}
	command := lookupCommand(cmdStr)
	if command == nil {
		c.AddReplyStr("-ERR: unknpwn command")
		resetClient(c)
		return
	}
	command.proc(c)
	resetClient(c)
}

// 释放 args refCount -1
func freeArgs(client *GodisClient) {
	// 从头节点一个一个删掉
	for _, arg := range client.args {
		arg.DecrRefCount()
	}
}

func freeReplyList(client *GodisClient) {
	for client.reply.length != 0 {
		n := client.reply.head
		client.reply.DelNode(n)
		n.Val.DecrRefCount()
	}
}

/*
释放客户端
1. 释放client的所有args,其实就是refCount减一
2. 从map中删除
3. 从事件循环中移除
4. 释放replyList
*/
func freeClient(client *GodisClient) {
	freeArgs(client)
	delete(server.clients, client.fd)
	server.keLoop.RemoveFileEvent(client.fd, KE_READABLE)
	server.keLoop.RemoveFileEvent(client.fd, KE_WRITABLE)
	freeReplyList(client)
	Close(client.fd)
}

func resetClient(client *GodisClient) {
	freeArgs(client)
	client.cmdType = COMMAND_UNKNOWN
	client.bulkNum = 0
	client.bulkLen = 0
}

// 找到结束的位置
// 如果没找到，就返回错误喽
func (client *GodisClient) findLineInQuery() (int, error) {
	index := strings.Index(string(client.queryBuf[:client.queryLen]), "\r\n")
	if index < 0 && client.queryLen > GODIS_MAX_INLINE {
		return index, errors.New("too long inline cmd")
	}
	return index, nil
}

/*
拿到bulk的数量
*/
func (client *GodisClient) getNumInQuery(s, e int) (int, error) {
	num, err := strconv.Atoi(string(client.queryBuf[s:e]))
	client.queryBuf = client.queryBuf[e+2:]
	client.queryLen -= e + 2
	return num, err
}

/*
处理inline格式的请求
从请求中把一个个参数拿出来就行了。
*/
func handleInlineBuf(client *GodisClient) (bool, error) {
	index, err := client.findLineInQuery()
	if index < 0 {
		return false, err
	}

	subs := strings.Split(string(client.queryBuf[:index]), " ") // inline是用空格分开的
	client.queryBuf = client.queryBuf[index+2:]                 // 往后走，除掉/r/n
	client.queryLen -= index + 2
	client.args = make([]*Gobj, len(subs))
	for i, sub := range subs {
		client.args[i] = CreateObject(GSTR, sub)
	}

	return true, nil
}

/*
处理bulk格式的请求
*/
func handleBulkBuf(client *GodisClient) (bool, error) {
	if client.bulkNum == 0 {
		index, err := client.findLineInQuery()
		if index < 0 {
			return false, err
		}

		if client.queryBuf[0] != '$' {
			return false, errors.New("except $ for bulk length") // $符号后面是bulk的长度
		}

		bnum, err := client.getNumInQuery(1, index)
		if err != nil {
			return false, err
		}
		if bnum == 0 {
			return true, nil
		}
		client.bulkNum = bnum
		client.args = make([]*Gobj, bnum)
	}
	// 读取每一个bulk
	for client.bulkNum > 0 {
		if client.bulkLen == 0 {
			index, err := client.findLineInQuery()
			if index < 0 {
				return false, err
			}

			if client.queryBuf[0] != '$' {
				return false, errors.New("expect $ for bulk length")
			}

			blen, err := client.getNumInQuery(1, index)
			if err != nil || blen == 0 {
				return false, err
			}
			if blen > GODIS_MAX_BULK {
				return false, errors.New("too big bulk")
			}
			client.bulkLen = blen
		}
		if client.queryLen < client.bulkLen+2 {
			return false, nil
		}
		index := client.bulkLen
		if client.queryBuf[index] != '\r' || client.queryBuf[index+1] != '\n' {
			return false, errors.New("expect CRLF for bulk end")
		}
		client.args[len(client.args)-client.bulkNum] = CreateObject(GSTR, string(client.queryBuf[:index]))
		client.queryBuf = client.queryBuf[index+2:]
		client.queryLen = 0
		client.bulkLen = 0
		client.bulkNum -= 1
	}
	return true, nil
}

/*
处理queryBuf
1. 判断使用的是哪种命令
2. 根据两种情况进行处理。
*/
func ProcessQueryBuf(client *GodisClient) error {
	for client.queryLen > 0 {
		if client.cmdType == COMMAND_UNKNOWN {
			if client.queryBuf[0] == '*' {
				client.cmdType = COMMAND_BULK
			} else {
				client.cmdType = COMMAND_INLINE
			}
		}
		var ok bool
		var err error
		if client.cmdType == COMMAND_BULK {
			ok, err = handleBulkBuf(client)
		} else if client.cmdType == COMMAND_INLINE {
			ok, err = handleInlineBuf(client)
		} else {
			return errors.New("unknown Godis Command Type")
		}
		if err != nil {
			return err
		}
		if ok {
			if len(client.args) == 0 {
				resetClient(client)
			} else {
				ProcessCommand(client)
			}
		} else {
			break
		}
	}
	return nil
}

/*
从client中读取请求
1. 判断queryBuf剩余大小够不够 GODIS_MAX_BULK 如果不够，就扩容
2.
*/
func ReadQueryFromClient(loop *KeLoop, fd int, extra interface{}) {
	client := extra.(*GodisClient)
	if len(client.queryBuf)-client.queryLen < GODIS_MAX_BULK {
		client.queryBuf = append(client.queryBuf, make([]byte, GODIS_MAX_BULK)...)
	}
	n, err := Read(fd, client.queryBuf[client.queryLen:])
	if err != nil {
		log.Printf("client %v read err: %v\n", fd, err)
		freeClient(client)
		return
	}
	client.queryLen += n
	log.Printf("read %v bytes from client:%v\n", n, client.fd)
	log.Printf("ReadRueryFromClient, queryBuf: %v\n", string(client.queryBuf))
	err = ProcessQueryBuf(client)
	if err != nil {
		log.Printf("process query buff error: %v\n", err)
		freeClient(client)
		return
	}
}

/*
发送回复到客户端
1.
*/
func SendReplyToClient(loop *KeLoop, fd int, extra interface{}) {
	client := extra.(*GodisClient)
	log.Printf("SendReplyToClient, reply len:%v\n", client.reply.Length())
	for client.reply.length > 0 {
		rep := client.reply.First()
		buf := []byte(rep.Val.StrVal())
		bufLen := len(buf)
		if client.sentLen < bufLen {
			n, err := Write(fd, buf[client.sentLen:])
			if err != nil {
				log.Printf("send reply err: %v\n", err)
				freeClient(client)
				return
			}
		}
	}
}

func StrEqual(a, b *Gobj) bool {
	if a.Type != GSTR || b.Type != GSTR {
		return false
	}
	return a.StrVal() == b.StrVal()
}

func StrHash(key *Gobj) int64 {
	if key.Type != GSTR {
		return 0
	}
	hash := fnv.New64()
	hash.Write([]byte(key.StrVal()))
	return int64(hash.Sum64())
}

/*
创建client
*/
func CreateClient(fd int) *GodisClient {
	var client GodisClient
	client.fd = fd
	client.db = server.db
	client.queryBuf = make([]byte, GODIS_IO_BUF)
	client.reply = ListCreate(ListType{EqualFunc: StrEqual})
	return &client
}

/*
*
接收连接的步骤:
1. Accept,从server fd中创建了client fd
2. 创建 client
3. 注册到 server.clients 这个map中
4. 注册 fileEvent
*/
func AcceptHandler(loop *KeLoop, fd int, extra interface{}) {
	cfd, err := Accept(fd)
	if err != nil {
		log.Printf("accept err: %v\n", err)
		return
	}
	client := CreateClient(fd)
	// 这里漏了，应该要检查最大连接数的
	server.clients[cfd] = client
	server.keLoop.AddFileEvent(cfd, KE_READABLE, ReadQueryFromClient, client)
	log.Printf("accept client,fd: %v\n", cfd)
}

/*
*
定时任务，每100ms跑一次
1. TODO
*/
func ServerCron(loop *KeLoop, fd int, extra interface{}) {

}

/*
*
1. 设置端口号
2. 创建clients 的map
3. 设置db，db中有两个Dict，每个Dict有两个函数：哈希和equal。
4. 创建事件循环
5. 创建tcp server
*/
func initServer(config *Config) error {
	server.port = config.Port
	server.clients = make(map[int]*GodisClient)
	server.db = &GodisDB{
		data:   DictCreate(DictType{HashFunc: StrHash, EqualFunc: StrEqual}),
		expire: DictCreate(DictType{HashFunc: StrHash, EqualFunc: StrEqual}),
	}
	var err error
	if server.keLoop, err = KeLoopCreate(); err != nil {
		return err
	}
	server.fd, err = TcpServer(server.port)
	return err
}

/*
*
1. 加载配置 其实就一个端口号
2. 初始化server
3. 添加fileEvent: AcceptHandler 用于接受连接
4. 添加timeEvent: ServerCron 用于检查过期
5. 开启事件循环
*/
func main() {
	// 启动的时候指定 配置文件路径
	path := os.Args[1]
	config, err := LoadConfig(path)
	if err != nil {
		log.Panicf("config error: %v\n", err)
	}
	err = initServer(config)
	if err != nil {
		log.Panicf("init server error: %v\n", err)
	}
	server.keLoop.AddFileEvent(server.fd, KE_READABLE, AcceptHandler, nil) // 注册文件事件，开始接受连接
	server.keLoop.AddTimeEvent(KE_NORMAL, 1000, ServerCron, nil)
	log.Printf("go-redis server started")
	server.keLoop.KeMain()
}
