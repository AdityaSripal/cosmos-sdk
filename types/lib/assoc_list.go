package lib

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	wire "github.com/cosmos/cosmos-sdk/wire"
)

type Scorable interface {
	Score() int64
}

type AssocList interface {
	Len() uint64

	// Operations with element
	Get(interface{}, *Scorable) error
	Set(interface{}, Scorable)
	Delete(interface{})

	// Operations with score
	Iterate(interface{}, *Scorable, func() bool)
	IterateScore(interface{}, *Scorable, func() bool)
	IterateRange(Scorable, Scorable, interface{}, *Scorable, func() bool)

	// Heap-like operations
	GetMin(interface{}, *Scorable)
	GetMax(interface{}, *Scorable)
	DeleteMin()
	DeleteMax()

	// Keys
	ScoreKey() []byte
	SortKey() []byte
}

type assocList struct {
	cdc   *wire.Codec
	store sdk.KVStore
	// ScoreKey: Element -> (Scorable, uint)
	// SortKey:  (Score, uint) -> Element
}

func (al assocList) Get(elem interface{}, ptr *Scorable) error {
	key := al.ScoreKey(elem)
	bz := al.store.Get(key)
	return al.cdc.UnmarshalBinary(bz, ptr)
}

func (al assocList) Set(elem interface{}, score Scorable) {
	key := al.ScoreKey(elem)
	bz := al.cdc.MustMarshalBinary(score)
	al.store.Set(key, bz)
}

func (al assocList) ScoreKey(elem interface{}) []byte {
	bz, err := al.cdc.MarshalBinary(elem)
	if err != nil {
		panic(err)
	}
	return append([]byte{0x00}, bz...)
}

func (al assocList) SortKey(score Scorable, index uint) []byte {
	return append(append([]byte{0x01}, []byte(fmt.Sprintf("%020d", score.Score()))), []byte(fmt.Sprintf("%010d", index)))
}
