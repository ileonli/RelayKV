package raft

import (
	"sync"
	"sync/atomic"
)

type State uint32

const (
	Candidate State = iota + 1

	Follower

	Leader

	Shutdown
)

func (s State) String() string {
	switch s {
	case Candidate:
		return "Candidate"
	case Follower:
		return "Follower"
	case Leader:
		return "Leader"
	case Shutdown:
		return "Shutdown"
	default:
		return "Unknown"
	}
}

type serverState struct {
	myself Server

	leaderID ServerID

	state State

	voteFor ServerID

	currentTerm uint64

	commitIndex uint64

	lastApplied uint64
}

func (s *serverState) me() ServerID {
	return s.myself.ServerID
}

func (s *serverState) getLeaderID() ServerID {
	v := (*uint64)(&s.leaderID)
	return ServerID(atomic.LoadUint64(v))
}

func (s *serverState) setLeaderID(id ServerID) {
	v := (*uint64)(&s.leaderID)
	atomic.StoreUint64(v, uint64(id))
}

func (s *serverState) getState() State {
	v := (*uint32)(&s.state)
	return State(atomic.LoadUint32(v))
}

func (s *serverState) setState(state State) {
	v := (*uint32)(&s.state)
	atomic.StoreUint32(v, uint32(state))
}

func (s *serverState) getVoteFor() ServerID {
	v := (*uint64)(&s.voteFor)
	return ServerID(atomic.LoadUint64(v))
}

func (s *serverState) setVoteFor(id ServerID) {
	v := (*uint64)(&s.voteFor)
	atomic.StoreUint64(v, uint64(id))
}

func (s *serverState) isVoted() bool {
	return s.getVoteFor() != NoneID
}

func (s *serverState) getCurrentTerm() uint64 {
	v := &s.currentTerm
	return atomic.LoadUint64(v)
}

func (s *serverState) setCurrentTerm(term uint64) {
	v := &s.currentTerm
	atomic.StoreUint64(v, term)
}

func (s *serverState) addCurrentTerm(extra uint64) {
	v := &s.currentTerm
	atomic.AddUint64(v, extra)
}

func (s *serverState) getCommitIndex() uint64 {
	v := &s.commitIndex
	return atomic.LoadUint64(v)
}

func (s *serverState) setCommitIndex(index uint64) {
	v := &s.commitIndex
	atomic.StoreUint64(v, index)
}

func (s *serverState) getLastApplied() uint64 {
	v := &s.lastApplied
	return atomic.LoadUint64(v)
}

func (s *serverState) setLastApplied(index uint64) {
	v := &s.lastApplied
	atomic.StoreUint64(v, index)
}

// ----------------------
//    followerLogState
// ----------------------

type followerLogState struct {
	mu sync.RWMutex

	nextIndex  []uint64
	matchIndex []uint64
}

func (p *followerLogState) getNextIndex(id ServerID) uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.nextIndex[id]
}

func (p *followerLogState) setNextIndex(id ServerID, index uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.nextIndex[id] = index
}

func (p *followerLogState) getMatchIndex(id ServerID) uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.matchIndex[id]
}

func (p *followerLogState) setMatchIndex(id ServerID, index uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.matchIndex[id] = index
}

func (p *followerLogState) getNextAndMatchIndex(id ServerID) (uint64, uint64) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.nextIndex[id], p.matchIndex[id]
}

func (p *followerLogState) setNextAndMatchIndex(id ServerID, nextIndex, matchIndex uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.nextIndex[id] = nextIndex
	p.matchIndex[id] = matchIndex
}

func (p *followerLogState) getBiggerOrEqualMatchNum(index uint64) uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	cnt := 0
	for _, matchIndex := range p.matchIndex {
		if matchIndex >= index {
			cnt++
		}
	}
	return uint64(cnt)
}
