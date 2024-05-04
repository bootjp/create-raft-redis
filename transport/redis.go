package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"raft-redis-cluster/raft"
	"raft-redis-cluster/store"
	"strings"
	"time"

	hraft "github.com/hashicorp/raft"
	"github.com/tidwall/redcon"
)

type Redis struct {
	listen net.Listener
	store  store.Store
	raft   *hraft.Raft
}

// NewRedis creates a new Redis transport.
func NewRedis(store store.Store, raft *hraft.Raft) *Redis {
	return &Redis{
		store: store,
		raft:  raft,
	}
}

func (r *Redis) Serve(addr string) error {
	var err error
	r.listen, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return r.handle()
}

func (r *Redis) handle() error {
	return redcon.Serve(r.listen,
		func(conn redcon.Conn, cmd redcon.Command) {
			fmt.Println("cmd", cmd)
			err := r.validateCmd(cmd)
			if err != nil {
				conn.WriteError(err.Error())
				return
			}
			r.processCmd(conn, cmd)
		},
		func(conn redcon.Conn) bool {
			return true
		},
		func(conn redcon.Conn, err error) {
			if err != nil {
				fmt.Println("error:", err)
			}
		},
	)
}

var argsLen = map[string]int{
	"GET": 2,
	"SET": 3,
	"DEL": 2,
}

const (
	commandName = 0
	keyName     = 1
	value       = 2
)

func (r *Redis) validateCmd(cmd redcon.Command) error {
	if len(cmd.Args) == 0 {
		return errors.New("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
	}

	if len(cmd.Args) < argsLen[string(cmd.Args[commandName])] {
		return errors.New("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
	}

	// check args length
	plainCmd := strings.ToUpper(string(cmd.Args[commandName]))

	if len(cmd.Args) != argsLen[plainCmd] {
		return errors.New("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
	}

	return nil
}

func (r *Redis) processCmd(conn redcon.Conn, cmd redcon.Command) {
	ctx := context.TODO()

	plainCmd := strings.ToUpper(string(cmd.Args[commandName]))
	switch plainCmd {
	case "GET":
		val, err := r.store.Get(ctx, cmd.Args[keyName])
		if err != nil {
			switch {
			case errors.Is(err, store.ErrKeyNotFound):
				conn.WriteNull()
				return
			default:
				conn.WriteError(err.Error())
				return
			}
		}
		conn.WriteBulk(val)
	case "SET":
		kvCmd := &raft.KVCmd{
			Op:  raft.Put,
			Key: cmd.Args[keyName],
			Val: cmd.Args[value],
		}

		b, err := json.Marshal(kvCmd)
		if err != nil {
			conn.WriteError(err.Error())
			return
		}

		f := r.raft.Apply(b, time.Second*1)
		if f.Error() != nil {
			conn.WriteError(f.Error().Error())
			return
		}

		conn.WriteString("OK")
	default:
		conn.WriteError("ERR unknown command '" + string(cmd.Args[0]) + "'")
	}
}

func (r *Redis) Close() error {
	return r.listen.Close()
}

func (r *Redis) Addr() net.Addr {
	return r.listen.Addr()
}
