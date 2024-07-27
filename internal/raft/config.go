package raft

// ConfigChange represents a configuration change request.
type ConfigChange struct {
	Type   string // "add" or "remove"
	NodeID string // ID of the node to add or remove
}

// JointConsensus represents a joint consensus request.
type JointConsensus struct {
	NewConfig []string
	OldConfig []string
}
