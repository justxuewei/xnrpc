package main

import (
	"github.com/xavier-niu/xnrpc/pkg/rpc/server"
	"log"
	"net"
)

func main() {
	l, err := net.Listen("tcp", ":10000")
	if err != nil {
		log.Fatal("network error: ", err)
	}
	log.Println("start rpc server on", l.Addr())
	for {
		server.Accept(l)
	}
}
