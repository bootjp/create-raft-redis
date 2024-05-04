package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"raft-redis-cluster/raft"
	"raft-redis-cluster/store"
	"raft-redis-cluster/transport"

	hraft "github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

var (
	myAddr        = flag.String("address", "localhost:50051", "TCP host+port for this node")
	redisAddr     = flag.String("redis_address", "localhost:6379", "TCP host+port for redis")
	raftId        = flag.String("raft_id", "", "Node id used by Raft")
	raftDir       = flag.String("raft_data_dir", "data/", "Raft data dir")
	raftBootstrap = flag.Bool("raft_bootstrap", false, "Whether to bootstrap the Raft cluster")
)

var configs map[string]struct {
	raftId    string
	raftDir   string
	myAddr    string
	redisAddr string
}

func init() {
	flag.Parse()
	validateFlags()
}

func validateFlags() {
	if *raftId == "" {
		log.Fatalf("flag --raft_id is required")
	}

	if *myAddr == "" {
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
	st := raft.NewStateMachine(store.NewMemoryStore())
	r, err := NewRaft(*raftDir, *raftId, *myAddr, st)
	if err != nil {
		log.Fatalln(err)
	}

	s := store.NewMemoryStore()
	redis := transport.NewRedis(s, r)
	fmt.Println(*redisAddr)
	fmt.Println("Redis server started")
	err = redis.Serve(*redisAddr)
	if err != nil {
		log.Fatalln(err)
	}
}

// snapshotRetainCount スナップショットの保持数
const snapshotRetainCount = 2

func NewRaft(basedir string, id string, address string, fsm hraft.FSM) (*hraft.Raft, error) {
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

	tm, err := hraft.NewTCPTransport(address, nil, 3, 10, os.Stderr)
	if err != nil {
		log.Fatalln(err)
	}

	r, err := hraft.NewRaft(c, fsm, ldb, sdb, fss, tm)
	if err != nil {
		return nil, err
	}

	if *raftBootstrap {
		cfg := hraft.Configuration{
			Servers: []hraft.Server{
				{
					Suffrage: hraft.Voter,
					ID:       hraft.ServerID(id),
					Address:  hraft.ServerAddress(address),
				},
			},
		}
		f := r.BootstrapCluster(cfg)
		if err := f.Error(); err != nil {
			return nil, err
		}
	}

	return r, nil
}
