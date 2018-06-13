package store

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type prefixStore struct {
	store  KVStore
	prefix []byte
}

// Implements Store
func (s prefixStore) GetStoreType() StoreType {
	return sdk.StoreTypePrefix
}

// Implements CacheWrap
func (s prefixStore) CacheWrap() CacheWrap {
	return NewCacheKVStore(s)
}

// Implements KVStore
func (s prefixStore) Get(key []byte) []byte {
	return s.store.Get(append(s.prefix, key...))
}

// Implements KVStore
func (s prefixStore) Has(key []byte) bool {
	return s.store.Has(append(s.prefix, key...))
}

// Implements KVStore
func (s prefixStore) Set(key, value []byte) {
	s.store.Set(append(s.prefix, key...), value)
}

// Implements KVStore
func (s prefixStore) Delete(key []byte) {
	s.store.Delete(append(s.prefix, key...))
}

// Implements KVStore
func (s prefixStore) Prefix(prefix string) KVStore {
	return prefixStore{s, []byte(prefix)}
}

// Implements KVStore
func (s prefixStore) Iterator(start, end []byte) Iterator {
	start = append(s.prefix, start...)
	end = append(s.prefix, end...)
	return s.store.Iterator(start, end)
}

// Implements KVStore
func (s prefixStore) ReverseIterator(start, end []byte) Iterator {
	start = append(s.prefix, start...)
	end = append(s.prefix, end...)
	return s.store.ReverseIterator(start, end)
}
