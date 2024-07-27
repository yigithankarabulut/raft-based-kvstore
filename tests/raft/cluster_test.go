package raft

import (
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/yigithankarabulut/raft/internal/raft"
)

func CreateCluster(n int) []*raft.Node {
	var peers []string
	for i := 0; i < n; i++ {
		peers = append(peers, "node"+strconv.Itoa(i))
	}

	peerChan := make(map[string]chan raft.RPCMessage)
	for _, peer := range peers {
		peerChan[peer] = make(chan raft.RPCMessage, 16)
	}

	nodes := make([]*raft.Node, 0)
	for _, peer := range peers {
		node := raft.NewNode(peer, peers, peerChan)
		nodes = append(nodes, node)
	}

	return nodes
}

func StartCluster(nodes []*raft.Node) {
	for _, node := range nodes {
		node.Start()
	}
}

func StopCluster(nodes []*raft.Node) {
	for _, node := range nodes {
		node.Stop()
	}
}

func FindLeader(nodes []*raft.Node) *raft.Node {
	for _, node := range nodes {
		if node.State() == raft.Leader {
			return node
		}
	}
	return nil
}

func TestCluster_SetCommand(t *testing.T) {
	nodes := CreateCluster(3)
	StartCluster(nodes)

	time.Sleep(2 * time.Second)

	leader := FindLeader(nodes)
	if leader == nil {
		t.Fatal("No leader found")
	}

	command := []byte("set x 10")
	if !leader.HandleClientRequest(command) {
		t.Fatal("Failed to handle client request")
	}

	time.Sleep(2 * time.Second)

	for _, node := range nodes {
		if node.State() != raft.Leader {
			found := false
			for _, entry := range node.Log() {
				if string(entry.Command) == "set x 10" {
					found = true
					log.Printf("[%s] Command replicated to node %+v\n", node.ID(), node.Log())
					break
				}
			}
			if !found {
				t.Fatal("Failed to replicate set x 10")
			}
		}
	}

	StopCluster(nodes)
}

func TestCluster_SetCommandMultiple(t *testing.T) {
	nodes := CreateCluster(3)
	StartCluster(nodes)

	time.Sleep(2 * time.Second)

	leader := FindLeader(nodes)
	if leader == nil {
		t.Fatal("No leader found")
	}

	command := []byte("set x 10")
	if !leader.HandleClientRequest(command) {
		t.Fatal("Failed to handle client request")
	}

	if !leader.HandleClientRequest([]byte("set x 15")) {
		t.Fatal("Failed to handle client request")
	}

	time.Sleep(2 * time.Second)

	for _, node := range nodes {
		if node.State() != raft.Leader {
			found := false
			for _, entry := range node.Log() {
				if string(entry.Command) == "set x 15" {
					found = true
					log.Printf("[%s] Command replicated to node %+v\n", node.ID(), node.Log())
					break
				}
			}
			if !found {
				t.Fatal("Failed to replicate set x 15")
			}
		}
	}

	StopCluster(nodes)
}

func TestCluster_SetCommandWithLeaderFailure(t *testing.T) {
	nodes := CreateCluster(3)
	StartCluster(nodes)

	time.Sleep(2 * time.Second)

	leader := FindLeader(nodes)
	if leader == nil {
		t.Fatal("No leader found")
	}

	command := []byte("set x 10")
	if !leader.HandleClientRequest(command) {
		t.Fatal("Failed to handle client request")
	}

	leader.Stop()

	leader = FindLeader(nodes)
	if leader == nil {
		t.Fatal("No leader found")
	}

	if !leader.HandleClientRequest([]byte("set x 15")) {
		t.Fatal("Failed to handle client request")
	}

	time.Sleep(2 * time.Second)

	for _, node := range nodes {
		if node.State() != raft.Leader {
			found := false
			for _, entry := range node.Log() {
				if string(entry.Command) == "set x 15" {
					found = true
					log.Printf("[%s] Command replicated to node %+v\n", node.ID(), node.Log())
					break
				}
			}
			if !found {
				t.Fatal("Failed to replicate set x 10 after leader failure")
			}
		}
	}

	StopCluster(nodes)
}
