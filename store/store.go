package store

import (
	"github.com/bootjp/go-kvlib/store"
)

type Store struct {
	store.Store
}

func NewStore() *Store {
	return &Store{
		Store: store.NewMemoryStore(),
	}
}
