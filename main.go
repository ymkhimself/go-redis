package main

import (
	"log"
	"os"
)

const (
	GODIS_IO_BUF   int = 1024 * 12
	GODIS_MAX_BULK int = 1024 * 4
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

}

func findKeyRead(keu *Gobj) *Gobj {
	return nil
}

func getCommand(c *GodisClient) {

}

func setCommand(c *GodisClient) {

}

func expireCommand(c *GodisClient) {

}

func lookupCommand(cmdStr string) *GodisCommand {
	return nil
}

func (c *GodisClient) AddReply(o *Gobj) {

}

func (c *GodisClient) AddReplyStr(str string) {

}

func ProcessCommand(c *GodisClient) {

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

}

func (client *GodisClient) findLineInQuery() (int, error) {

	return 0, nil
}

func (client *GodisClient) getNumInQuery(s, e int) (int, error) {

}

func handleInlineBuf(client *GodisClient) (bool, error) {

}

func handleBulkBuf(client *GodisClient) (bool, error) {

}

func ProcessQueryBuf(client *GodisClient) error {

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

func SendReplyToClient(loop *KeLoop, fd int, extra interface{}) {

}

func StrEqual(a, b *Gobj) bool {

}

func StrHash(key *Gobj) int64 {

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
