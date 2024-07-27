package raft

import "log"

// LogEntry represents a log entry.
type LogEntry struct {
	Term    uint64
	Index   uint64
	Command []byte
}

// LogReplicationResult represents a log replication result.
type LogReplicationResult struct {
	successCount int    // successCount is the number of successful replications.
	term         uint64 // term is the term of the leader.
}

// Snapshot represents a snapshot of the log entries.
type Snapshot struct {
	LastIncludedIndex uint64
	LastIncludedTerm  uint64
	Data              []byte
}

// InstallSnapshotArgs represents the arguments for installing a snapshot.
type InstallSnapshotArgs struct {
	Term              uint64
	LeaderID          string
	LastIncludedIndex uint64
	LastIncludedTerm  uint64
	Offset            uint64
	Data              []byte
	Done              bool
}

// InstallSnapshotReply represents the reply for installing a snapshot.
type InstallSnapshotReply struct {
	Term uint64
}

func (n *Node) startReplication() {
	for _, peer := range n.peers {
		if peer != n.id {
			go n.replicateLog(peer)
		}
	}
}

func (n *Node) replicateLog(peer string) {
	for {
		var (
			prevLogIndex uint64
			prevLogTerm  uint64
		)

		if n.nextIndex[peer] == 0 {
			n.nextIndex[peer] = 1
		}

		prevLogIndex = n.nextIndex[peer] - 1
		if prevLogIndex > 0 {
			prevLogTerm = n.log[prevLogIndex].Term
		}

		entries := n.log[n.nextIndex[peer]:]
		args := &AppendEntriesArgs{
			Term:         n.currentTerm,
			LeaderID:     n.id,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			Entries:      entries,
			LeaderCommit: n.commitIndex,
		}

		var reply AppendEntriesReply
		if err := n.sendRPC(peer, AppendEntriesRPC, args, &reply); err != nil {
			log.Printf("[%s] Error replicating log to %s: %v\n", n.id, peer, err)
			return
		}

		if reply.Success {
			n.nextIndex[peer] = prevLogIndex + uint64(len(entries)) + 1
			n.matchIndex[peer] = n.nextIndex[peer] - 1
			return
		}
		if reply.Term > n.currentTerm {
			n.becomeFollower(reply.Term)
			return
		}
		n.nextIndex[peer]--
	}
}
