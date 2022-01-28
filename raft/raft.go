package raft

import "sync"

type Raft struct {
	mu sync.RWMutex

	serverState      *serverState
	followerLogState *followerLogState

	cluster Cluster

	logger Logger
	trans  Transport

	logs    EntryStore
	persist PersistStore

	shutdownCh <-chan struct{}
}

func (r *Raft) becomeFollower() {
	r.serverState.setState(Follower)
	r.serverState.setLeaderID(NoneID)
	r.serverState.setVoteFor(NoneID)
}

func (r *Raft) becomeCandidate() {
	r.serverState.setState(Candidate)
	r.serverState.setLeaderID(NoneID)
	r.serverState.setVoteFor(r.serverState.me())
}

func (r *Raft) becomeLeader() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.serverState.setState(Leader)
	r.serverState.setVoteFor(NoneID)
	r.serverState.setLeaderID(NoneID)

	lastIndex, _ := r.logs.LastIndex()
	r.cluster.visit(func(s Server) {
		r.followerLogState.setNextAndMatchIndex(
			s.ServerID, lastIndex+1, 0)
	})
}

func (r *Raft) step() {
	for {
		switch r.serverState.getState() {
		case Follower:
			r.stepFollower()
		case Candidate:
			r.stepCandidate()
		case Leader:
			r.stepLeader()
		case Shutdown:
			r.logger.Warningf("%v shutdown", r.serverState.server)
			return
		}
	}
}

func (r *Raft) stepFollower() {
	for r.serverState.getState() == Follower {
		select {
		case <-r.shutdownCh:
			return
		}
	}
}

func (r *Raft) stepCandidate() {
	for r.serverState.getState() == Candidate {
		select {
		case <-r.shutdownCh:
			return
		}
	}
}

func (r *Raft) stepLeader() {
	for r.serverState.getState() == Leader {
		select {
		case <-r.shutdownCh:
			return
		}
	}
}
