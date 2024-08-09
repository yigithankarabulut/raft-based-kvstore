package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/yigithankarabulut/raft/internal/raft"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		panic("Usage: raftnode <node_id>")
	}

	node := "node"
	m := make(map[string]string)

	for i := 1; i < len(args); i++ {
		m[node+args[i]] = "localhost" + args[i]
	}

	for k, v := range m {
		fmt.Printf("Key: %s, Value: %s\n", k, v)
	}

	nodeID := "node" + args[1]

	n := raft.NewNode(nodeID, m)

	if n == nil {
		panic("Failed to create node")
	}

	if err := n.Start(); err != nil {
		panic(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	n.Stop()

	time.Sleep(1 * time.Second)

}
