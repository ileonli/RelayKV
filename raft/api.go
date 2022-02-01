package raft

import (
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
		commitCh:         make(chan struct{}),
		shutdownCh:       make(chan struct{}),
	}

	go rf.step()
	go rf.runFSM()

	return rf
}
