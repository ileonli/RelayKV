package raft

import "RelayKV/raft/pb"

func (r *Raft) getPrevLog(id ServerID) (uint64, uint64) {
	nextIndex := r.followerLogState.getNextIndex(id)
	if nextIndex <= 1 {
		return 0, 0
	}
	entry, err := r.logStore.GetEntry(nextIndex - 1)
	if err != nil {
		panic(err)
	}
	if entry == nil {
		return 0, 0
	}
	return entry.Index, entry.Term
}

func (r *Raft) getLastLog() (uint64, uint64) {
	entry, err := r.logStore.LastEntry()
	if err != nil {
		panic(err)
	}
	if entry == nil {
		return 0, 0
	}
	return entry.Index, entry.Term
}

func (r *Raft) getEntries(index uint64) []*pb.Entry {
	var entries []*pb.Entry
	lastIndex, err := r.logStore.LastIndex()
	if err != nil {
		panic(err)
	}
	for i := index; i <= lastIndex; i++ {
		entry, err := r.logStore.GetEntry(i)
		if err != nil {
			panic(err)
		}
		entries = append(entries, entry)
	}
	return entries
}
