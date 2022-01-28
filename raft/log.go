package raft

func (r *Raft) getPrevLog(id ServerID) (uint64, uint64) {
	nextIndex := r.followerLogState.getNextIndex(id)
	if nextIndex <= 1 {
		return 0, 0
	}
	entry, err := r.logs.GetEntry(nextIndex - 1)
	if err != nil {
		panic(err)
	}
	if entry == nil {
		return 0, 0
	}
	return entry.Index, entry.Term
}

func (r *Raft) getLastLog() (uint64, uint64) {
	entry, err := r.logs.LastEntry()
	if err != nil {
		panic(err)
	}
	if entry == nil {
		return 0, 0
	}
	return entry.Index, entry.Term
}

func (r *Raft) getEntries(index uint64) []Entry {
	var entries []Entry
	lastIndex, err := r.logs.LastIndex()
	if err != nil {
		panic(err)
	}
	for i := index; i <= lastIndex; i++ {
		entry, err := r.logs.GetEntry(i)
		if err != nil {
			panic(err)
		}
		entries = append(entries, *entry)
	}
	return entries
}
