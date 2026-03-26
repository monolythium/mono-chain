package testutil

import (
	"context"
	"errors"

	corestore "cosmossdk.io/core/store"
)

// FailSetStore is a KVStore that returns an error on Set. Reads return
// nil (not found) so collections handles them gracefully via ErrNotFound.
type FailSetStore struct{}

func (FailSetStore) Get([]byte) ([]byte, error)                                 { return nil, nil }
func (FailSetStore) Has([]byte) (bool, error)                                   { return false, nil }
func (FailSetStore) Set([]byte, []byte) error                                   { return errors.New("store write failed") }
func (FailSetStore) Delete([]byte) error                                        { return nil }
func (FailSetStore) Iterator([]byte, []byte) (corestore.Iterator, error)        { return nil, nil }
func (FailSetStore) ReverseIterator([]byte, []byte) (corestore.Iterator, error) { return nil, nil }

// FailSetStoreService is a KVStoreService that always returns a FailSetStore.
type FailSetStoreService struct{}

func (FailSetStoreService) OpenKVStore(context.Context) corestore.KVStore {
	return FailSetStore{}
}
