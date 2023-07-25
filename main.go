package main

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

func ReadQueryFromClient(loop *KeLoop, fd int, extra interface{}) {

}

func main() {
}
