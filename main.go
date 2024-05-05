package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"raft-redis-cluster/raft"
	"raft-redis-cluster/store"
	"raft-redis-cluster/transport"
	"strings"
	"time"

	hraft "github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

// initialPeersList nodeID->address mapping
type initialPeersList map[string]string

func (i *initialPeersList) Set(value string) error {
	node := strings.Split(value, ",")
	for _, n := range node {
		parts := strings.Split(n, "=")
		if len(parts) != 2 {
			return errors.New("invalid peer format. expected nodeID=address")
		}
		(*i)[parts[0]] = parts[1]
	}
	return nil
}

func (i *initialPeersList) String() string {
	return fmt.Sprintf("%v", *i)
}

var (
	raftAddr     = flag.String("address", "localhost:50051", "TCP host+port for this raft node")
	redisAddr    = flag.String("redis_address", "localhost:6379", "TCP host+port for redis")
	raftId       = flag.String("raft_id", "", "Node id used by Raft")
	raftDir      = flag.String("raft_data_dir", "", "Raft data dir")
	initialPeers = initialPeersList{}
)

func init() {
	flag.Var(&initialPeers, "initial_peers", "Initial peers for the Raft cluster")
	flag.Parse()
	validateFlags()
}

func validateFlags() {
	if *raftId == "" {
		log.Fatalf("flag --raft_id is required")
	}

	if *raftAddr == "" {
		log.Fatalf("flag --address is required")
	}

	if *redisAddr == "" {
		log.Fatalf("flag --redis_address is required")
	}

	if *raftDir == "" {
		log.Fatalf("flag --raft_data_dir is required")
	}
}

func main() {
	datastore := store.NewMemoryStore()
	st := raft.NewStateMachine(datastore)
	r, err := NewRaft(*raftDir, *raftId, *raftAddr, st, initialPeers)
	if err != nil {
		log.Fatalln(err)
	}

	redis := transport.NewRedis(datastore, r)
	err = redis.Serve(*redisAddr)
	if err != nil {
		log.Fatalln(err)
	}
}

// snapshotRetainCount スナップショットの保持数
const snapshotRetainCount = 2

func NewRaft(basedir string, id string, address string, fsm hraft.FSM, nodes initialPeersList) (*hraft.Raft, error) {
	c := hraft.DefaultConfig()
	c.LocalID = hraft.ServerID(id)

	baseDir := filepath.Join(basedir, id)

	ldb, err := raftboltdb.NewBoltStore(filepath.Join(baseDir, "logs.dat"))
	if err != nil {
		return nil, err
	}

	sdb, err := raftboltdb.NewBoltStore(filepath.Join(baseDir, "stable.dat"))
	if err != nil {
		return nil, err
	}

	fss, err := hraft.NewFileSnapshotStore(baseDir, snapshotRetainCount, os.Stderr)
	if err != nil {
		return nil, err
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}

	tm, err := hraft.NewTCPTransport(address, tcpAddr, 10, time.Second*10, os.Stderr)
	if err != nil {
		return nil, err
	}

	r, err := hraft.NewRaft(c, fsm, ldb, sdb, fss, tm)
	if err != nil {
		return nil, err
	}

	cfg := hraft.Configuration{
		Servers: []hraft.Server{
			{
				Suffrage: hraft.Voter,
				ID:       hraft.ServerID(id),
				Address:  hraft.ServerAddress(address),
			},
		},
	}

	// 自分以外の initialPeers を追加
	for name, addr := range nodes {
		cfg.Servers = append(cfg.Servers, hraft.Server{
			Suffrage: hraft.Voter,
			ID:       hraft.ServerID(name),
			Address:  hraft.ServerAddress(addr),
		})
	}

	f := r.BootstrapCluster(cfg)
	if err := f.Error(); err != nil {
		return nil, err
	}

	return r, nil
}
