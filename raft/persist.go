package raft

import (
	"strconv"
	"sync"
)

type PersistStore interface {
	Set(key []byte, val []byte) error

	// Get returns the value for key, or an empty byte slice if key was not found.
	Get(key []byte) ([]byte, error)

	SetUint64(key []byte, val uint64) error

	// GetUint64 returns the uint64 value for key, or 0 if key was not found.
	GetUint64(key []byte) (uint64, error)
}

type InMemoryPersistStore struct {
	mu sync.RWMutex

	maps map[string]string
}

func NewInMemoryPersistStore() *InMemoryPersistStore {
	return &InMemoryPersistStore{
		maps: make(map[string]string),
	}
}

func (s *InMemoryPersistStore) Set(key []byte, val []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.maps[string(key)] = string(val)
	return nil
}

func (s *InMemoryPersistStore) Get(key []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return []byte(s.maps[string(key)]), nil
}

func (s *InMemoryPersistStore) SetUint64(key []byte, val uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.maps[string(key)] = strconv.FormatUint(val, 10)
	return nil
}

func (s *InMemoryPersistStore) GetUint64(key []byte) (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	parseUint, err := strconv.ParseUint(
		s.maps[string(key)], 10, 64)
	if err != nil {
		return 0, err
	}
	return parseUint, nil
}
