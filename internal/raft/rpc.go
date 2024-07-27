package raft

import (
	"fmt"
	"log"
)

// AppendEntriesArgs represents the arguments for appending entries to the log
type AppendEntriesArgs struct {
	Term         uint64
	LeaderID     string
	PrevLogIndex uint64
	PrevLogTerm  uint64
	Entries      []LogEntry
	LeaderCommit uint64
}

// AppendEntriesReply represents the reply for appending entries to the log
type AppendEntriesReply struct {
	Term    uint64
	Success bool
}

// RequestVoteArgs represents the arguments for requesting a vote
type RequestVoteArgs struct {
	Term         uint64
	CandidateID  string
	LastLogIndex uint64
	LastLogTerm  uint64
}

// RequestVoteReply represents the reply for requesting a vote
type RequestVoteReply struct {
	Term        uint64
	VoteGranted bool
}

func (n *Node) sendRPC(peer string, rpc string, args interface{}, reply interface{}) error {
	msg := RPCMessage{
		rpc:   rpc,
		args:  args,
		reply: make(chan interface{}),
	}

	n.PeerChan[peer] <- msg
	res := <-msg.reply

	switch rpc {
	case RequestVoteRPC:
		*reply.(*RequestVoteReply) = res.(RequestVoteReply)
	case AppendEntriesRPC:
		*reply.(*AppendEntriesReply) = res.(AppendEntriesReply)
	default:
		return fmt.Errorf("unknown RPC: %s", rpc) //nolint:err113
	}

	return nil
}

func (n *Node) handleRPC(msg RPCMessage) {
	log.Printf("[%s] Handling RPC: %s\n", n.id, msg.rpc)
	switch msg.rpc {
	case RequestVoteRPC:
		args := msg.args.(RequestVoteArgs)
		reply := RequestVoteReply{}
		if err := n.RequestVote(args, &reply); err != nil {
			log.Printf("[%s] Error handling RequestVote RPC: %v\n", n.id, err)
		}
		msg.reply <- reply
	case AppendEntriesRPC:
		var args AppendEntriesArgs
		switch v := msg.args.(type) {
		case *AppendEntriesArgs:
			args = *v
		case AppendEntriesArgs:
			args = v
		default:
			panic(fmt.Sprintf("unexpected type %T", msg.args))
		}
		reply := AppendEntriesReply{}
		if err := n.AppendEntries(args, &reply); err != nil {
			log.Printf("[%s] Error handling AppendEntries RPC: %v\n", n.id, err)
		}
		msg.reply <- reply
	default:
		log.Printf("[%s] Unknown RPC: %s\n", n.id, msg.rpc)
	}
}

// RequestVote is called by candidates to request votes from other nodes in the cluster.
func (n *Node) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	if args.Term < n.currentTerm {
		reply.Term = n.currentTerm
		reply.VoteGranted = false
		return nil
	}

	if args.Term > n.currentTerm {
		n.becomeFollower(args.Term)
	}
	if n.votedFor == "" || n.votedFor == args.CandidateID &&
		n.isLogUpToDate(args.LastLogIndex, args.LastLogTerm) {
		reply.VoteGranted = true
		n.votedFor = args.CandidateID
	} else {
		reply.VoteGranted = false
	}

	reply.Term = n.currentTerm
	return nil
}

// AppendEntries is called by the leader to replicate log entries on followers.
func (n *Node) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	reply.Success = false
	reply.Term = n.currentTerm

	if args.Term < n.currentTerm {
		return nil
	}

	if args.Term > n.currentTerm {
		n.becomeFollower(args.Term)
	}

	n.resetElectionTimer()

	if args.PrevLogIndex > uint64(len(n.log)) {
		return nil
	}

	if args.PrevLogIndex > 0 && (len(n.log) <= int(args.PrevLogIndex) || n.log[args.PrevLogIndex-1].Term != args.PrevLogTerm) {
		return nil
	}

	for i, entry := range args.Entries {
		index := args.PrevLogIndex + uint64(i) + 1
		if index <= uint64(len(n.log)) {
			if n.log[index-1].Term != entry.Term {
				n.log = n.log[:index-1]
				n.log = append(n.log, entry)
			}
		} else {
			n.log = append(n.log, entry)
		}
	}

	if args.LeaderCommit > n.commitIndex {
		n.commitIndex = min(args.LeaderCommit, uint64(len(n.log)))
	}

	reply.Success = true
	return nil
}
