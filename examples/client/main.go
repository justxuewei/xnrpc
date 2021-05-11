package main

import (
	"fmt"
	c "github.com/xavier-niu/xnrpc/pkg/rpc/client"
	"log"
	"sync"
)

func main() {
	client, _ := c.Dial("tcp", ":10000")
	defer func() { _ = client.Close() }()

	wg := new(sync.WaitGroup)
	for i:=0; i<5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("geerpc req %d", i)
			var reply string
			if err := client.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Println("reply:", reply)
		}(i)
	}
	wg.Wait()
}
