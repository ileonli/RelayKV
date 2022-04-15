package raft

import (
	"RelayKV/raft/pb"
	"encoding/json"
	"time"
)

func NewRaft(
	applyCh chan *ApplyMsg, c *Configuration,
	logger Logger, trans Transport,
	logStore EntryStore, persistStore PersistStore) *Raft {

	verifyConfig(c)

	serverState := &serverState{
		myself:      c.Me,
		leaderID:    c.Me.ServerID,
		state:       Follower,
		voteFor:     NoneID,
		currentTerm: 0,
		commitIndex: 0,
		lastApplied: 0,
	}

	size := c.Cluster.size()
	followerLogState := &followerLogState{
		nextIndex:  make([]uint64, size),
		matchIndex: make([]uint64, size),
	}

	rf := &Raft{
		serverState:      serverState,
		followerLogState: followerLogState,
		cluster:          c.Cluster,
		logger:           logger,
		trans:            trans,
		logStore:         logStore,
		persistStore:     persistStore,
		electronTimer:    time.NewTimer(0),
		electronTimeout:  c.ElectronTimeout,
		heartbeatTimeout: c.HeartbeatTimeout,
		rpcCh:            trans.Consume(),
		applyCh:          applyCh,
		commitCh:         make(chan struct{}, 1),
		shutdownCh:       make(chan struct{}),
	}

	go rf.step()
	go rf.runFSM()

	return rf
}

func (r *Raft) GetState() State {
	return r.getState()
}

func (r *Raft) GetTerm() uint64 {
	return r.getCurrentTerm()
}

func (r *Raft) GetLeaderId() uint64 {
	return uint64(r.getLeaderID())
}

func (r *Raft) Start(command interface{}) (uint64, uint64, bool) {
	isLeader := r.getState() == Leader
	if !isLeader {
		return 0, 0, false
	}

	bytes, err := json.Marshal(command)
	if err != nil {
		panic(err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	term := r.getCurrentTerm()
	lastIndex, _ := r.getLastLog()
	entry := &pb.Entry{Index: lastIndex + 1, Term: term, Data: bytes}
	if err := r.logStore.StoreEntry(entry); err != nil {
		panic(err)
	}
	r.followerLogState.setNextIndex(r.me().ServerID, entry.Index+1)
	r.followerLogState.setNextAndMatchIndex(r.me().ServerID, entry.Index+1, entry.Index)
	return entry.Index, entry.Term, isLeader
}
