package store

import hraft "github.com/hashicorp/raft"

var prefixRedisAddr = []byte("___redisAddr")

func GetRedisAddrByNodeID(store hraft.StableStore, lid hraft.ServerID) (string, error) {
	v, err := store.Get(append(prefixRedisAddr, []byte(lid)...))
	if err != nil {
		return "", err
	}

	return string(v), nil
}

func SetRedisAddrByNodeID(store hraft.StableStore, lid hraft.ServerID, addr string) error {
	return store.Set(append(prefixRedisAddr, []byte(lid)...), []byte(addr))
}
