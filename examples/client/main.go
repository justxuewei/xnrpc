package main

import (
	"encoding/json"
	"fmt"
	"github.com/xavier-niu/xnrpc/pkg/rpc"
	"github.com/xavier-niu/xnrpc/pkg/rpc/codec"
	"log"
	"net"
)

func main() {
	conn, _ := net.Dial("tcp", "127.0.0.1:10000")
	defer func() { _ = conn.Close() }()

	_ = json.NewEncoder(conn).Encode(rpc.DefaultOption)
	cc := codec.NewGobCodec(conn)
	// send request & receive response
	for i:=0; i<5; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq: uint64(i),
		}

		_ = cc.Write(h, fmt.Sprintf("geerpc req %d", h.Seq))
		_ = cc.ReadHeader(h)
		var reply string
		_ = cc.ReadBody(&reply)
		log.Println("reply: ", reply)
	}
}
