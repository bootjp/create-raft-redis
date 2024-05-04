package raft

import (
	"io"

	"github.com/hashicorp/raft"
)

var _ raft.FSMSnapshot = (*KVSnapshot)(nil)

type KVSnapshot struct {
	io.ReadWriter
}

func (f *KVSnapshot) Persist(sink raft.SnapshotSink) error {
	defer sink.Close()
	_, err := io.Copy(sink, f)
	return err
}

func (f *KVSnapshot) Release() {
}
