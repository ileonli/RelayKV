package raft

// --------------------------
//    SendRequestVote RPC
// --------------------------

type RequestVoteRequest struct {
	Term        uint64
	CandidateId ServerID

	LastLogIndex uint64
	LastLogTerm  uint64
}

type RequestVoteResponse struct {
	Term        uint64
	VoteGranted bool
}

// ---------------------------
//    SendAppendEntries RPC
// ---------------------------

type AppendEntriesRequest struct {
	Term     uint64
	LeaderId ServerID

	PrevLogIndex uint64
	PrevLogTerm  uint64
	LeaderCommit uint64
	Entries      []Entry
}

type AppendEntriesResponse struct {
	Term          uint64
	Success       bool
	ConflictIndex uint64
}
