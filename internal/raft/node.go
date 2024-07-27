package raft

import (
	"log"
	"math/rand"
	"time"
)

// NodeState represents the state of a node.
type NodeState int

const (
	// HeartbeatTimeout represents the timeout for sending heartbeats.
	HeartbeatTimeout = 50 * time.Millisecond

	// RequestVoteRPC represents the request vote RPC.
	RequestVoteRPC = "RequestVote"

	// AppendEntriesRPC represents the append entries RPC.
	AppendEntriesRPC = "AppendEntries"
)

const (
	// Follower represents the follower state.
	Follower NodeState = iota

	// Candidate represents the candidate state.
	Candidate

	// Leader represents the leader state.
	Leader
)

// RPCMessage represents an RPC message to be sent.
type RPCMessage struct {
	rpc   string
	args  interface{}
	reply chan interface{}
}

// ElectionResult represents the result of an election.
type ElectionResult struct {
	votes       int
	totalVotes  int
	term        uint64
	wonElection bool
}

// Node represents a node in the Raft cluster
type Node struct {
	id              string
	state           NodeState
	currentTerm     uint64
	votedFor        string
	log             []LogEntry
	commitIndex     uint64
	lastApplied     uint64
	nextIndex       map[string]uint64
	matchIndex      map[string]uint64
	peers           []string
	electionTimeout time.Duration
	PeerChan        map[string]chan RPCMessage
	doneCh          chan struct{}
}

// NewNode creates a new Raft node with the given ID, peers, and peer channels
func NewNode(id string, peers []string, peerChan map[string]chan RPCMessage) *Node {
	n := &Node{
		id:              id,
		state:           Follower,
		currentTerm:     0,
		votedFor:        "",
		log:             []LogEntry{{0, 0, nil}},
		commitIndex:     0,
		lastApplied:     0,
		nextIndex:       make(map[string]uint64),
		matchIndex:      make(map[string]uint64),
		peers:           peers,
		electionTimeout: time.Duration(rand.Intn(150)+150) * time.Millisecond, //nolint:gosec
		PeerChan:        peerChan,
		doneCh:          make(chan struct{}),
	}

	for _, peer := range peers {
		n.nextIndex[peer] = 1
		n.matchIndex[peer] = 0
	}

	n.resetElectionTimer()

	return n
}

// Start starts the Raft node
func (n *Node) Start() {
	log.Printf("[%s] Starting node\n", n.id)
	go func() {
		for {
			select {
			case <-time.After(n.electionTimeout):
				if n.state != Leader {
					n.startElection()
				}
			case msg := <-n.PeerChan[n.id]:
				n.handleRPC(msg)

			case <-n.doneCh:
				return
			}
		}
	}()
}

// Stop stops the Raft node
func (n *Node) Stop() {
	log.Printf("[%s] Stopping node\n", n.id)
	n.doneCh <- struct{}{}
}

// State returns the state of the node
func (n *Node) State() NodeState {
	return n.state
}

// Log returns the log of the node
func (n *Node) Log() []LogEntry {
	return n.log
}

// ID returns the ID of the node
func (n *Node) ID() string {
	return n.id
}

// HandleClientRequest handles a client request
func (n *Node) HandleClientRequest(command []byte) bool {
	if n.state != Leader {
		return false
	}

	success := n.appendEntries(command)
	if success {
		log.Printf("[%s] Append entries: %s\n", n.id, command)
		n.startReplication()
	}

	return success
}

func (n *Node) resetElectionTimer() {
	log.Printf("[%s] Resetting election timer\n", n.id)
	n.electionTimeout = time.Duration(rand.Intn(150)+150) * time.Millisecond //nolint:gosec
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
				log.Printf("[%s] Stopping heartbeat (not a leader)\n", n.id)
				return
			}

			for _, peer := range n.peers {
				if peer == n.id {
					continue
				}

				log.Printf("[%s] Sending heartbeat to %s\n", n.id, peer)

				go func(peer string) {
					args := &AppendEntriesArgs{
						Term:         n.currentTerm,
						LeaderID:     n.id,
						PrevLogIndex: n.nextIndex[peer] - 1,
						PrevLogTerm:  n.log[n.nextIndex[peer]-1].Term,
						Entries:      []LogEntry{},
						LeaderCommit: n.commitIndex,
					}

					var reply AppendEntriesReply
					if err := n.sendRPC(peer, AppendEntriesRPC, *args, &reply); err != nil {
						log.Printf("[%s] Error sending heartbeat to %s: %v\n", n.id, peer, err)
						return
					}

					if reply.Term > n.currentTerm {
						log.Printf("[%s] Received higher term from %s\n", n.id, peer)
						n.becomeFollower(reply.Term)
					}
				}(peer)

			}
			time.Sleep(HeartbeatTimeout)
		}
	}
}

func (n *Node) startElection() {
	log.Printf("[%s] Want to start election (current term: %d)\n", n.id, n.currentTerm)
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

	args := &RequestVoteArgs{
		Term:         n.currentTerm,
		CandidateID:  n.id,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}

	// Request votes from peers
	for _, peer := range n.peers {
		if peer == n.id {
			continue
		}

		go func(peer string) {
			var reply RequestVoteReply
			log.Printf("[%s] Requesting vote from %s\n", n.id, peer)
			if err := n.sendRPC(peer, RequestVoteRPC, *args, &reply); err != nil {
				log.Printf("[%s] Error requesting vote from %s: %v\n", n.id, peer, err)
				return
			}

			if reply.VoteGranted {
				votesReceived++
				if votesReceived > len(n.peers)/2 {
					if n.state == Candidate {
						log.Printf("[%s] Won election\n", n.id)
						n.becomeLeader()
					}
				}
			} else if reply.Term > n.currentTerm {
				log.Printf("[%s] Received higher term from %s\n", n.id, peer)
				n.becomeFollower(reply.Term)
			}
		}(peer)
	}
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

	go n.startHeartbeat()
}

func (n *Node) becomeCandidate() {
	log.Printf("[%s] Becoming candidate\n", n.id)
	n.state = Candidate
}

func (n *Node) appendEntries(command []byte) bool {
	if n.state != Leader {
		return false
	}

	log.Printf("[%s] Appending entries\n", n.id)

	entry := LogEntry{
		Term:    n.currentTerm,
		Index:   n.log[len(n.log)-1].Index + 1,
		Command: command,
	}

	n.log = append(n.log, entry)

	// Send entries to peers
	for _, peer := range n.peers {
		if peer == n.id {
			continue
		}

		go n.sendEntries(peer)
	}

	return true
}

func (n *Node) sendEntries(peer string) {
	prevLogIndex := n.nextIndex[peer] - 1
	prevLogTerm := n.log[prevLogIndex].Term

	args := &AppendEntriesArgs{
		Term:         n.currentTerm,
		LeaderID:     n.id,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      n.log[prevLogIndex+1:],
		LeaderCommit: n.commitIndex,
	}

	var reply AppendEntriesReply
	if err := n.sendRPC(peer, AppendEntriesRPC, *args, &reply); err != nil {
		log.Printf("[%s] Error sending entries to %s: %v\n", n.id, peer, err)
		return
	}

	if reply.Success {
		n.nextIndex[peer] = uint64(len(n.log))
		n.matchIndex[peer] = n.nextIndex[peer] - 1
		n.updateCommitIndex()
	} else {
		if reply.Term > n.currentTerm {
			log.Printf("[%s] Received higher term from %s\n", n.id, peer)
			n.becomeFollower(reply.Term)
		} else {
			n.nextIndex[peer]--
		}
	}
}

func (n *Node) updateCommitIndex() {
	for i := n.commitIndex + 1; i <= uint64(len(n.log)); i++ {
		if n.log[i-1].Term == n.currentTerm {
			matchCount := 1
			for _, peer := range n.peers {
				if n.matchIndex[peer] >= i {
					matchCount++
				}
			}
			if matchCount > len(n.peers)/2 {
				n.commitIndex = i
				go n.applyCommittedEntries()
			}
		}
	}
}

func (n *Node) applyCommittedEntries() {
	for i := n.lastApplied + 1; i <= n.commitIndex; i++ {
		entry := n.log[i-1]
		log.Printf("[%s] Applying log entry %d: %v\n", n.id, i, entry)
		n.lastApplied = i
	}
}

func (n *Node) isLogUpToDate(lastLogIndex, lastLogTerm uint64) bool {
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
