package raft

import (
	"errors"
	"fmt"
	"sync"
)

// Entry used to store log.
type Entry struct {
	Index uint64
	Term  uint64
	Date  interface{}
}

// EntryStore is used to provide an interface for storing
// and retrieving logs in a durable fashion.
type EntryStore interface {
	// FirstIndex returns the first Entry's Index. 0 for no entries.
	FirstIndex() (uint64, error)

	// LastIndex returns the last Entry's Index. 0 for no entries.
	LastIndex() (uint64, error)

	// FirstEntry returns the first Entry. nil for no entries.
	FirstEntry() (*Entry, error)

	// LastEntry returns the last Entry. nil for no entries.
	LastEntry() (*Entry, error)

	// GetEntry gets a log entry at a given index.
	GetEntry(index uint64) (*Entry, error)

	// StoreEntry stores a log entry.
	StoreEntry(entry *Entry) error

	// StoreEntries stores multiple log entries.
	StoreEntries(entries []*Entry) error

	// DeleteRange deletes a range of log entries. The range is [min, max].
	// If min == max, delete the entry at min index.
	DeleteRange(min, max uint64) error
}

type InMemoryEntryStore struct {
	mu sync.RWMutex

	entries []*Entry
}

func NewInMemoryEntryStore() *InMemoryEntryStore {
	return &InMemoryEntryStore{
		entries: []*Entry{
			{Index: 0, Term: 0, Date: nil},
		},
	}
}

func (s *InMemoryEntryStore) FirstIndex() (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 0 for none Entry, we just return the first Entry's index
	return s.entries[0].Index, nil
}

func (s *InMemoryEntryStore) LastIndex() (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	n := len(s.entries)
	return s.entries[n-1].Index, nil
}

func (s *InMemoryEntryStore) FirstEntry() (*Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if entriesLen := len(s.entries); entriesLen == 1 {
		return nil, nil
	}
	return s.entries[1], nil
}

func (s *InMemoryEntryStore) LastEntry() (*Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entriesLen := len(s.entries)
	if entriesLen == 1 {
		return nil, nil
	}
	return s.entries[entriesLen-1], nil
}

func (s *InMemoryEntryStore) GetEntry(index uint64) (*Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entriesLen := uint64(len(s.entries))
	if 0 >= index || entriesLen <= index {
		return nil, errors.New(fmt.Sprintf(
			"index: %v out of bound", index))
	}
	return s.entries[index], nil
}

func (s *InMemoryEntryStore) StoreEntry(entry *Entry) error {
	return s.StoreEntries([]*Entry{entry})
}

func (s *InMemoryEntryStore) StoreEntries(entries []*Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append(s.entries, entries...)
	return nil
}

func (s *InMemoryEntryStore) DeleteRange(min, max uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if min <= 0 || max <= 0 {
		return errors.New(fmt.Sprintf(
			"illegal index of min: %v, max: %v", min, max))
	}

	if min > max {
		return errors.New(fmt.Sprintf(
			"index of min: %v > max: %v", min, max))
	}

	entriesLen := uint64(len(s.entries))
	if max >= entriesLen {
		return errors.New(fmt.Sprintf(
			"index: %v out of bound", max))
	}
	if min == max {
		max = min + 1
	} else if max == entriesLen-1 {
		max = entriesLen
	} else {
		max = max + 1
	}
	s.entries = append(s.entries[:min], s.entries[max:]...)

	return nil
}
