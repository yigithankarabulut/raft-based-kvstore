package raft

import (
	"context"
	"errors"

	"github.com/yigithankarabulut/raft/internal/pb"
	"google.golang.org/grpc"
)

func (n *Node) sendRequestVoteRPC(ctx context.Context, peer string, args *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	id, ok := n.peers.Load(peer)
	if !ok {
		return nil, errors.New("peer not found")
	}
	conn, err := grpc.Dial(id.(string), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			return
		}
	}(conn)

	return pb.NewRaftServiceClient(conn).RequestVote(ctx, args)
}

func (n *Node) sendAppendEntriesRPC(ctx context.Context, peer string, args *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	id, ok := n.peers.Load(peer)
	if !ok {
		return nil, errors.New("peer not found")
	}
	conn, err := grpc.Dial(id.(string), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			return
		}
	}(conn)

	return pb.NewRaftServiceClient(conn).AppendEntries(ctx, args)
}

func (n *Node) RequestVote(ctx context.Context,
	req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	var res pb.RequestVoteResponse

	if req.Term < n.currentTerm {
		res = pb.RequestVoteResponse{
			Term:        n.currentTerm,
			VoteGranted: false,
		}
		return &res, nil
	}

	if req.Term > n.currentTerm {
		n.becomeFollower(req.Term)
	}

	if n.votedFor == "" || n.votedFor == req.CandidateId &&
		n.isLogUpToDate(req.LastLogIndex, req.LastLogTerm) {
		res = pb.RequestVoteResponse{
			VoteGranted: true,
		}
		n.votedFor = req.CandidateId
		n.resetElectionTimer()
	} else {
		res = pb.RequestVoteResponse{
			VoteGranted: false,
		}
	}

	res.Term = n.currentTerm
	return &res, nil
}

func (n *Node) AppendEntries(ctx context.Context,
	req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	var res pb.AppendEntriesResponse

	res.Success = false
	res.Term = n.currentTerm

	if req.Term < n.currentTerm {
		return &res, nil
	}

	if req.Term > n.currentTerm {
		n.becomeFollower(req.Term)
	}

	n.resetElectionTimer()

	if req.PrevLogIndex > uint64(len(n.log)) {
		return &res, nil
	}

	if req.PrevLogIndex > 0 {
		if req.PrevLogIndex >= uint64(len(n.log)-1) {
			res.Success = false
			res.Term = n.currentTerm
			return &res, nil
		}
		if n.log[req.PrevLogIndex].Term != req.PrevLogTerm {
			res.Success = false
			res.Term = n.currentTerm
			// if the logs don't match, truncate the log
			n.log = n.log[:req.PrevLogIndex]
			return &res, nil
		}
	}

	// Append new entries
	for i, entry := range req.Entries {
		if uint64(i)+req.PrevLogIndex+1 < uint64(len(n.log)) {
			if n.log[req.PrevLogIndex+uint64(i)+1].Term != entry.Term {
				n.log = n.log[:req.PrevLogIndex+uint64(i)+1]
				break
			}
		}
		n.log = append(n.log, pb.LogEntry{
			Term:    entry.Term,
			Index:   entry.Index,
			Command: entry.Command,
		})
	}

	if req.LeaderCommit > n.commitIndex {
		n.commitIndex = min(req.LeaderCommit, uint64(len(n.log)))
	}

	res.Success = true
	return &res, nil
}
