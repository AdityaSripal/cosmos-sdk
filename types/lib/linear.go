package lib

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	wire "github.com/cosmos/cosmos-sdk/wire"
)

// Linear defines a primitive mapper type
type Linear struct {
	cdc   *wire.Codec
	store sdk.KVStore
}

// List is a Linear interface that provides list-like functions
// It panics when the element type cannot be (un/)marshalled by the codec
type List interface {

	// Len() returns the length of the list
	// The length is only increased by Push() and not decreased
	// List dosen't check if an index is in bounds
	// The user should check Len() before doing any actions
	Len() uint64

	// Get() returns the element by its index
	Get(uint64, interface{}) error

	// Set() stores the element to the given position
	// Setting element out of range will break length counting
	// Use Push() instead of Set() to append a new element
	Set(uint64, interface{})

	// Delete() deletes the element in the given position
	// Other elements' indices are preserved after deletion
	// Panics when the index is out of range
	Delete(uint64)

	// Push() inserts the element to the end of the list
	// It will increase the length when it is called
	Push(interface{})

	// Iterate*() is used to iterate over all existing elements in the list
	// Return true in the continuation to break
	// The second element of the continuation will indicate the position of the element
	// Using it with Get() will return the same one with the provided element

	// CONTRACT: No writes may happen within a domain while iterating over it.
	Iterate(interface{}, func(uint64) bool)
}

// NewList constructs new List
func NewList(cdc *wire.Codec, store sdk.KVStore) List {
	return Linear{
		cdc:   cdc,
		store: store,
	}
}

// Key for the length of the list
func LengthKey() []byte {
	return []byte{0x00}
}

// Key for the elements of the list
func ElemKey(index uint64) []byte {
	return append([]byte{0x01}, []byte(fmt.Sprintf("%020d", index))...)
}

// Len implements List
func (m Linear) Len() uint64 {
	bz := m.store.Get(LengthKey())
	if bz == nil {
		zero, err := m.cdc.MarshalBinary(0)
		if err != nil {
			panic(err)
		}
		m.store.Set(LengthKey(), zero)
		return 0
	}
	var res uint64
	if err := m.cdc.UnmarshalBinary(bz, &res); err != nil {
		panic(err)
	}
	return res
}

// Get implements List
func (m Linear) Get(index uint64, ptr interface{}) error {
	bz := m.store.Get(ElemKey(index))
	return m.cdc.UnmarshalBinary(bz, ptr)
}

// Set implements List
func (m Linear) Set(index uint64, value interface{}) {
	bz, err := m.cdc.MarshalBinary(value)
	if err != nil {
		panic(err)
	}
	m.store.Set(ElemKey(index), bz)
}

// Delete implements List
func (m Linear) Delete(index uint64) {
	m.store.Delete(ElemKey(index))
}

// Push implements List
func (m Linear) Push(value interface{}) {
	length := m.Len()
	m.Set(length, value)
	m.store.Set(LengthKey(), marshalUint64(m.cdc, length+1))
}

// IterateRead implements List
func (m Linear) Iterate(ptr interface{}, fn func(uint64) bool) {
	iter := sdk.KVStorePrefixIterator(m.store, []byte{0x01})
	for ; iter.Valid(); iter.Next() {
		v := iter.Value()
		if err := m.cdc.UnmarshalBinary(v, ptr); err != nil {
			panic(err)
		}
		k := iter.Key()
		s := string(k[len(k)-20:])
		index, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			panic(err)
		}
		if fn(index) {
			break
		}
	}

	iter.Close()
}

// Queue is a Linear interface that provides queue-like functions
// It panics when the element type cannot be (un/)marshalled by the codec
type Queue interface {
	// Push() inserts the elements to the rear of the queue
	Push(interface{})

	// Popping/Peeking on an empty queue will cause panic
	// The user should check IsEmpty() before doing any actions

	// Peek() returns the element at the front of the queue without removing it
	Peek(interface{}) error

	// Pop() returns the element at the front of the queue and removes it
	Pop()

	// IsEmpty() checks if the queue is empty
	IsEmpty() bool

	// Flush() removes elements it processed
	// Return true in the continuation to break
	// The interface{} is unmarshalled before the continuation is called
	// Starts from the top(head) of the queue
	// CONTRACT: Pop() or Push() should not be performed while flushing
	Flush(interface{}, func() bool)
}

// NewQueue constructs new Queue
func NewQueue(cdc *wire.Codec, store sdk.KVStore) Queue {
	return Linear{
		cdc:   cdc,
		store: store,
	}
}

// Key for the top element position in the queue
func TopKey() []byte {
	return []byte{0x02}
}

func (m Linear) getTop() (res uint64) {
	bz := m.store.Get(TopKey())
	if bz == nil {
		m.store.Set(TopKey(), marshalUint64(m.cdc, 0))
		return 0
	}

	if err := m.cdc.UnmarshalBinary(bz, &res); err != nil {
		panic(err)
	}

	return
}

func (m Linear) setTop(top uint64) {
	bz := marshalUint64(m.cdc, top)
	m.store.Set(TopKey(), bz)
}

// Peek implements Queue
func (m Linear) Peek(ptr interface{}) error {
	top := m.getTop()
	return m.Get(top, ptr)
}

// Pop implements Queue
func (m Linear) Pop() {
	top := m.getTop()
	m.Delete(top)
	m.setTop(top + 1)
}

// IsEmpty implements Queue
func (m Linear) IsEmpty() bool {
	top := m.getTop()
	length := m.Len()
	return top >= length
}

// Flush implements Queue
func (m Linear) Flush(ptr interface{}, fn func() bool) {
	top := m.getTop()
	length := m.Len()

	var i uint64
	for i = top; i < length; i++ {
		m.Get(i, ptr)
		m.Delete(i)
		if fn() {
			break
		}
	}
	m.setTop(i)
}

func marshalUint64(cdc *wire.Codec, i uint64) []byte {
	bz, err := cdc.MarshalBinary(i)
	if err != nil {
		panic(err)
	}
	return bz
}

func subspace(prefix []byte) (start, end []byte) {
	end = make([]byte, len(prefix))
	copy(end, prefix)
	end[len(end)-1]++
	return prefix, end
}
