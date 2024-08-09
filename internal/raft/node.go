package raft

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/yigithankarabulut/raft/internal/pb"
	"google.golang.org/grpc"
)

type NodeState int

const (
	HeartbeatTimeout = 300 * time.Millisecond
	ContextTimeout   = 50 * time.Millisecond
)

const (
	Follower NodeState = iota
	Candidate
	Leader
)

type Node struct {
	pb.UnimplementedRaftServiceServer
	id              string
	state           NodeState
	currentTerm     uint64
	votedFor        string
	log             []pb.LogEntry
	commitIndex     uint64
	lastApplied     uint64
	nextIndex       sync.Map
	matchIndex      sync.Map
	peers           sync.Map
	electionTimeout time.Duration
	doneCh          chan struct{}
	mu              sync.Mutex
	grpcServer      *grpc.Server
}

func NewNode(id string, peers map[string]string) *Node {
	n := &Node{
		id:              id,
		state:           Follower,
		currentTerm:     0,
		votedFor:        "",
		log:             []pb.LogEntry{{Term: 0, Index: 0}},
		commitIndex:     0,
		lastApplied:     0,
		nextIndex:       sync.Map{},
		matchIndex:      sync.Map{},
		peers:           sync.Map{},
		electionTimeout: 0,
		doneCh:          make(chan struct{}),
		mu:              sync.Mutex{},
	}
	for k, v := range peers {
		n.peers.Store(k, v)
		n.nextIndex.Store(k, uint64(1))
		n.matchIndex.Store(k, uint64(0))
	}

	n.resetElectionTimer()

	return n
}

func (n *Node) Stop() {
	log.Printf("[%s] Stopping node\n", n.id)
	n.doneCh <- struct{}{}
}

// Start starts the Raft node
func (n *Node) Start() error {
	log.Printf("[%s] Starting node\n", n.id)
	v, _ := n.peers.Load(n.id)
	log.Printf("listening on %s\n", v)

	lis, err := net.Listen("tcp", v.(string))
	if err != nil {
		return err
	}

	n.grpcServer = grpc.NewServer()
	pb.RegisterRaftServiceServer(n.grpcServer, n)

	go func() {
		if err := n.grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	time.Sleep(3 * time.Second)
	go n.run()

	return nil
}

func (n *Node) run() {
	for {
		select {
		case <-n.doneCh:
			log.Printf("[%s] Stopping node\n", n.id)
			n.grpcServer.Stop()
			return
		case <-time.After(n.electionTimeout):
			if n.state == Follower {
				log.Printf("[%s] Starting election\n", n.id)
				go n.startElection()
			}
		}
	}
}

func (n *Node) resetElectionTimer() {
	log.Printf("[%s] Resetting election timer\n", n.id)
	n.electionTimeout = time.Duration(rand.Intn(450)+450) * time.Millisecond
}

func (n *Node) syncMapLen(m *sync.Map) int {
	var i int
	m.Range(func(_, _ interface{}) bool {
		i++
		return true
	})
	return i
}

func (n *Node) startElection() {
	log.Printf("[%s] Want to start election (current term: %d)\n", n.id, n.currentTerm)
	n.mu.Lock()
	defer n.mu.Unlock()
	n.state = Candidate
	n.currentTerm++
	n.votedFor = n.id
	votesReceived := 1

	n.becomeCandidate()

	lastLogIndex := uint64(0)
	lastLogTerm := uint64(0)
	if len(n.log) > 0 {
		lastLogIndex = n.log[len(n.log)-1].Index
		lastLogTerm = n.log[len(n.log)-1].Term
	}

	req := pb.RequestVoteRequest{
		Term:         n.currentTerm,
		CandidateId:  n.id,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}

	n.peers.Range(func(k, v interface{}) bool {
		if k.(string) == n.id {
			return true
		}

		go func(peer string) {
			log.Printf("[%s] Requesting vote from %s\n", n.id, peer)
			ctx := context.Background()
			res, err := n.sendRequestVoteRPC(ctx, peer, &req)
			if err != nil {
				log.Printf("[%s] Error requesting vote from %s %v\n", n.id, peer, err)
				return
			}
			if res.VoteGranted {
				votesReceived++
				if votesReceived > n.syncMapLen(&n.peers)/2 {
					if n.state == Candidate {
						log.Printf("[%s] Won election\n", n.id)
						n.becomeLeader()
					}
				}
			} else if res.Term > n.currentTerm {
				log.Printf("[%s] Received higher term from %s\n", n.id, peer)
				n.becomeFollower(res.Term)
			}
		}(k.(string))
		return true
	})
}

func (n *Node) becomeFollower(term uint64) {
	log.Printf("[%s] Becoming follower\n", n.id)
	n.state = Follower
	n.currentTerm = term
	n.votedFor = ""
	n.resetElectionTimer()
}

func (n *Node) becomeLeader() {
	log.Printf("[%s] Becoming leader\n", n.id)
	n.state = Leader

	if err := n.appendEntries([]byte("")); err != nil {
		log.Printf("[%s] Error appending entries: %v\n", n.id, err)
	}

	n.peers.Range(func(k, v interface{}) bool {
		if k.(string) != n.id {
			n.nextIndex.Store(k, uint64(len(n.log)))
			n.matchIndex.Store(k, uint64(0))
		}
		return true
	})

	go n.startHeartbeat()
}

func (n *Node) becomeCandidate() {
	log.Printf("[%s] Becoming candidate\n", n.id)
	n.state = Candidate
}

func (n *Node) startHeartbeat() {
	log.Printf("[%s] Starting heartbeat\n", n.id)
	for {
		select {
		case <-n.doneCh:
			log.Printf("[%s] Stopping heartbeat\n", n.id)
			return
		default:
			if n.state != Leader {
				log.Printf("[%s] Stopping heartbeat (not a leader).\n", n.id)
				return
			}

			n.peers.Range(func(k, v interface{}) bool {
				if k.(string) == n.id {
					return true
				}

				log.Printf("[%s] Sending heartbeat to %s\n", n.id, k.(string))
				go func(peer string) {
					k, _ := n.nextIndex.Load(peer)
					log.Printf("[%s] Next index for %s %d\n", n.id, peer, k.(uint64))
					prevLogIndex := k.(uint64) - 1
					prevLogTerm := n.log[prevLogIndex].Term
					req := pb.AppendEntriesRequest{
						Term:         n.currentTerm,
						LeaderId:     n.id,
						PrevLogIndex: prevLogIndex,
						PrevLogTerm:  prevLogTerm,
						Entries:      nil,
						LeaderCommit: n.commitIndex,
					}

					ctx := context.Background()
					res, err := n.sendAppendEntriesRPC(ctx, peer, &req)
					if err != nil {
						log.Printf("[%s] Error sending heartbeat to %s: %v\n", n.id, peer, err)
						return
					}

					if res.Term > n.currentTerm {
						log.Printf("[%s] Received higher term from %s\n", n.id, peer)
						n.becomeFollower(res.Term)
					}
				}(k.(string))

				time.Sleep(HeartbeatTimeout)

				return true
			})
		}
	}
}

func (n *Node) appendEntries(command []byte) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state != Leader {
		return errors.New("not a leader")
	}

	log.Printf("[%s] Appending entries.\n", n.id)

	entry := pb.LogEntry{
		Term:    n.currentTerm,
		Index:   n.log[len(n.log)-1].Index + 1,
		Command: command,
	}

	n.log = append(n.log, entry)

	n.peers.Range(func(k, v interface{}) bool {
		if k.(string) == n.id {
			return true
		}

		go n.sendEntries(k.(string))

		return true
	})

	return nil
}

func (n *Node) sendEntries(peer string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	k, _ := n.nextIndex.Load(peer)

	prevLogIndex := k.(uint64) - 1
	if prevLogIndex >= uint64(len(n.log)) || prevLogIndex < 0 {
		log.Printf("[%s] prevLogIndex out of range: %d\n", n.id, prevLogIndex)
		return
	}

	prevLogTerm := n.log[prevLogIndex].Term
	if k.(uint64) > uint64(len(n.log)) {
		log.Printf("[%s] nextIndex out of range: %d\n", n.id, k.(uint64))
		return
	}

	entries := n.log[k.(uint64):]

	req := pb.AppendEntriesRequest{
		Term:         n.currentTerm,
		LeaderId:     n.id,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		LeaderCommit: n.commitIndex,
	}

	for _, entry := range entries {
		req.Entries = append(req.Entries, &pb.LogEntry{
			Term:    entry.Term,
			Index:   entry.Index,
			Command: entry.Command,
		})
	}

	ctx := context.Background()

	res, err := n.sendAppendEntriesRPC(ctx, peer, &req)
	if err != nil {
		log.Printf("[%s] Error sending entries to %s: %v\n", n.id, peer, err)
		return
	}

	if res.Success {
		n.nextIndex.Store(peer, uint64(len(n.log)))
		v, _ := n.nextIndex.Load(peer)
		n.matchIndex.Store(peer, v.(uint64)-1)
		n.updateCommitIndex()
	} else {
		if res.Term > n.currentTerm {
			log.Printf("[%s] Received higher term from %s\n", n.id, peer)
			n.becomeFollower(res.Term)
		} else {
			v, _ := n.nextIndex.Load(peer)
			n.nextIndex.Store(peer, v.(uint64)-1)
		}
	}
}

func (n *Node) updateCommitIndex() {
	n.mu.Lock()
	defer n.mu.Unlock()

	var matchIndexes []uint64
	n.matchIndex.Range(func(_, v interface{}) bool {
		matchIndexes = append(matchIndexes, v.(uint64))
		return true
	})

	sort.Slice(matchIndexes, func(i, j int) bool {
		return matchIndexes[i] < matchIndexes[j]
	})

	n.commitIndex = matchIndexes[len(matchIndexes)/2]

	log.Printf("[%s] Commit index updated: %d\n", n.id, n.commitIndex)
}

func (n *Node) isLogUpToDate(lastLogIndex, lastLogTerm uint64) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	var index int
	if len(n.log) != 0 {
		index = len(n.log) - 1
	}

	if lastLogTerm > n.log[index].Term {
		return true
	}

	if lastLogTerm == n.log[index].Term && lastLogIndex >= n.log[index].Index {
		return true
	}

	return false
}
