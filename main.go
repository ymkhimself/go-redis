package main

import (
	"log"
	"os"
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

func freeArgs(client *GodisClient) {

}

func freeReplyList(client *GodisClient) {

}

func freeClient(c *GodisClient) {

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

func ReadQueryFromClient(loop *KeLoop, fd int, extra interface{}) {

}

func SendReplyToClient(loop *KeLoop, fd int, extra interface{}) {

}

func StrEqual(a, b *Gobj) bool {

}

func StrHash(key *Gobj) int64 {

}

func CreateClient(fd int) *GodisClient {

}

func AcceptHandler(loop *KeLoop, fd int, extra interface{}) {

}

func ServerCron(loop *KeLoop, fd int, extra interface{}) {

}

func initServer(config *Config) error {
	server.port = config.Port
	server.clients = make(map[int]*GodisClient)
	server.db = &GodisDB{
		data: 
	}
}

func main() {
	// 启动的时候指定 配置文件路径
	path := os.Args[1]
	config, err := LoadConfig(path)
	if err != nil {
		log.Printf("config error: %v\n", err)
	}
	err = initServer(config)
	if err != nil {
		log.Printf("init server error: %v\n", err)
	}
	server.keLoop.AddFileEvent(server.fd, KE_READABLE, AcceptHandler, nil) // 注册文件事件，开始接受连接
	server.keLoop.AddTimeEvent(KE_NORMAL, 1000, ServerCron, nil)
	log.Printf("go-redis server started")
	server.keLoop.KeMain()
}
