package raft

import (
	"encoding/json"
	"io"

	"github.com/hashicorp/raft"
)

var _ raft.FSM = (*StateMachine)(nil)

type Op int

const (
	Put Op = iota
	Del
)

type cmd struct {
	Op  Op     `json:"op"`
	Key string `json:"key"`
	Val string `json:"val"`
}

type StateMachine struct {
}

// Apply applies a Raft log entry to the key-value store.
func (fsm *StateMachine) Apply(log *raft.Log) interface{} {
	c := &cmd{}

	err := json.Unmarshal(log.Data, c)
	if err != nil {
		return err
	}

	err = fsm.handleRequest(c)
	return err
}

// Snapshot returns a snapshot of the key-value store.

// Restore stores the key-value store to a previous state.
func (fsm *StateMachine) Restore(rc io.ReadCloser) error {
	return nil
}

// Snapshot returns a snapshot of the key-value store.
func (fsm *StateMachine) Snapshot() (raft.FSMSnapshot, error) {
	return nil, nil
}

func (fsm *StateMachine) handleRequest(cmd *cmd) error {
	return nil
}
