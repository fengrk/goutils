package profileutils

import (
	"github.com/frkhit/logger"
	"net/http"
	_ "net/http/pprof"
	"strconv"
)

// todo warning: use global var `DefaultServeMux`
type PProfServer struct {
	addr   string
	port   int
	server *http.Server
}

func NewPProfSever(addr string, port int) *PProfServer {
	server := &PProfServer{addr: addr, port: port,}
	server.Start()
	return server
}

func (server *PProfServer) Start() {
	if server.server != nil {
		logger.Warningln("server start before, no need to start it again!")
		return
	}
	
	server.server = &http.Server{Addr: server.addr + ":" + strconv.Itoa(server.port), Handler: nil}
	if err := server.server.ListenAndServe(); err != nil {
		logger.Fatal(err)
	}
	logger.Infof("profile server start, connect http://%s:%d/debug/pprof/\n", server.addr, server.port)
}

func (server *PProfServer) Close() {
	if server.server == nil {
		logger.Warningln("server close before, no need to close it again!")
		return
	}
	
	if err := server.server.Close(); err != nil {
		logger.Fatalf("fail to close server: %s", err)
	}
	server.server = nil
}

func (server *PProfServer) AddHandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	http.HandleFunc(pattern, handler)
}

func StartGolangPProf() *PProfServer {
	return NewPProfSever("127.0.0.1", 9001)
}
