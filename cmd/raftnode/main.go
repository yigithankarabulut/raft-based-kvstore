package main

//
// import (
//	"log"
//	"time"
//
//	"github.com/yigithankarabulut/raft/internal/raft"
// )
//
// func main() {
//	//peers := []string{"node1", "node2", "node3", "node4", "node5"}
//	peers := []string{"node1", "node2", "node3"}
//	peerChan := make(map[string]chan raft.RPCMessage)
//	for _, peer := range peers {
//		peerChan[peer] = make(chan raft.RPCMessage, 16)
//	}
//
//	node1 := raft.NewNode("node1", peers, peerChan)
//	node2 := raft.NewNode("node2", peers, peerChan)
//	node3 := raft.NewNode("node3", peers, peerChan)
//	//node4 := raft.NewNode("node4", peers, peerChan)
//	//node5 := raft.NewNode("node5", peers, peerChan)
//
//	//nodes := []*raft.Node{node1, node2, node3, node4, node5}
//
//	nodes := []*raft.Node{node1, node2, node3}
//
//	for _, node := range nodes {
//		node.Start()
//	}
//
//	// Wait for the election to complete
//	time.Sleep(2 * time.Second)
//
//	// Find the leader
//	var leader *raft.Node
//	for _, node := range nodes {
//		if node.State() == raft.Leader {
//			leader = node
//			break
//		}
//	}
//
//	if leader == nil {
//		panic("No leader found")
//	}
//
//	// Send a command to the leader
//	command := []byte("set x 10")
//	success := leader.HandleClientRequest(command)
//	if !success {
//		panic("Failed to handle client request")
//	}
//
//	if !leader.HandleClientRequest([]byte("get x")) {
//		log.Fatalf("Failed to handle client request")
//	}
//
//	// Wait for the command to be replicated
//	time.Sleep(2 * time.Second)
//
//	// Verify that the command is replicated to the followers
//	for _, node := range nodes {
//		if node.State() != raft.Leader {
//			found := false
//			for _, entry := range node.Log() {
//				if string(entry.Command) == "set x 10" {
//					found = true
//					log.Printf("------------------------------------------\n")
//					log.Printf("Command replicated to node %s\n", node.ID())
//					log.Printf("Log: %+v\n", node.Log())
//					log.Printf("------------------------------------------\n")
//					break
//				}
//			}
//			if !found {
//				log.Fatalf("Command not replicated to node %s", node.ID())
//			}
//		}
//	}
// }
