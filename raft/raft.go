package raft

import (
	"sync"
	"time"
)

type Raft struct {
	mu sync.RWMutex

	*serverState
	followerLogState *followerLogState

	cluster Cluster

	logger Logger
	trans  Transport

	logs    EntryStore
	persist PersistStore

	electronTimer *time.Timer

	electronTimeout, heartbeatTimeout time.Duration

	rpcCh      <-chan *RPC
	applyCh    chan *ApplyMsg
	commitCh   chan struct{}
	shutdownCh <-chan struct{}
}

func (r *Raft) becomeFollower() {
	r.setState(Follower)
	r.setLeaderID(NoneID)
	r.setVoteFor(NoneID)
}

func (r *Raft) becomeCandidate() {
	r.serverState.setState(Candidate)
	r.serverState.setLeaderID(NoneID)
	r.serverState.setVoteFor(r.serverState.me())
}

func (r *Raft) becomeLeader() {
	r.setState(Leader)
	r.setVoteFor(NoneID)
	r.setLeaderID(NoneID)

	lastIndex, _ := r.logs.LastIndex()
	r.cluster.visit(func(s Server) {
		r.followerLogState.setNextAndMatchIndex(
			s.ServerID, lastIndex+1, 0)
	}, false)
}

func (r *Raft) step() {
	for {
		switch r.getState() {
		case Follower:
			r.stepFollower()
		case Candidate:
			r.stepCandidate()
		case Leader:
			r.stepLeader()
		case Shutdown:
			r.logger.Warningf("%v shutdown", r.myself)
			return
		}
	}
}

func (r *Raft) stepFollower() {

	r.resetElectronTimer()

	for r.getState() == Follower {
		select {
		case rpc := <-r.rpcCh:
			r.processRPC(rpc)

		case <-r.electronTimer.C:
			r.logger.Infof("%v electron timeout, become Follower", r.me())
			r.becomeCandidate()

		case <-r.shutdownCh:
			return
		}
	}
}

func (r *Raft) stepCandidate() {

	// vote self, so have one cnt by default
	cnt, quorum := 1, int(r.cluster.quorum())
	r.logger.Infof("%v start campaign", r.me())
	voteResponses := r.campaign()

	timer := time.After(RandomTimeout(r.electronTimeout))

	for r.getState() == Candidate {
		select {
		case rpc := <-r.rpcCh:
			r.processRPC(rpc)

		case campaignResp := <-voteResponses:
			resp := campaignResp.resp

			r.logger.Warningf("%v receive vote reply: %v from: %v", r.me(), resp, campaignResp.id)

			if r.getCurrentTerm() > resp.Term {
				r.logger.Warningf("%v get smaller term: %d, ignore the resp: %v",
					r.me(), resp.Term, resp)
				continue
			}

			if r.getCurrentTerm() < resp.Term {
				r.logger.Warningf("%v get bigger term: %d, become Follower", r.me(), resp.Term)
				r.becomeFollower()
				r.setCurrentTerm(resp.Term)
			}

			if resp.VoteGranted {
				cnt++
			}

			if cnt > quorum {
				r.logger.Infof("%v get majority votes: %d, become Leader", r.me(), cnt)
				r.becomeLeader()
			}

		case <-timer:
			r.logger.Warningf("campaign timeout, restart")
			return

		case <-r.shutdownCh:
			return
		}
	}
}

func (r *Raft) stepLeader() {

	heartbeatResponses := make(chan *heartbeatRes, r.cluster.size()*3)

	tick := time.After(0)

	for r.getState() == Leader {
		select {
		case rpc := <-r.rpcCh:
			r.processRPC(rpc)

		case heartbeatResp := <-heartbeatResponses:
			id := heartbeatResp.id
			req := heartbeatResp.req
			resp := heartbeatResp.resp

			r.logger.Infof("%v receive heartbeat request: %v response: %v from id: %v",
				r.me(), req, resp, id)

			if r.getState() != Leader {
				r.logger.Warningf("%v is not a leader, ignore the appendEntries resp: %v", r.me(), resp)
				return
			}

			if r.getCurrentTerm() > resp.Term {
				r.logger.Warningf("%v reject out of date resp: %v", r.me(), resp)
				continue
			}

			if r.getCurrentTerm() < resp.Term {
				r.logger.Warningf("%v appendEntries get bigger term, become Follower", r.me())
				r.becomeFollower()
				r.setCurrentTerm(resp.Term)
			}

			// leader's log is not same to the Follower
			if !resp.Success {
				r.logger.Warningf("%v log is not same to the id: %d, back log to: %v",
					r.me(), id, resp.ConflictIndex)
				r.followerLogState.setNextIndex(id, resp.ConflictIndex)
				continue
			}

			// just for heartbeat
			if len(req.Entries) == 0 {
				continue
			}

			if resp.Success {

				r.mu.Lock()

				n := uint64(len(req.Entries))
				newMatchIndex := req.PrevLogIndex + n
				r.followerLogState.setNextAndMatchIndex(id, newMatchIndex+1, newMatchIndex)

				r.logger.Infof("%v success replicate to id: %v", r.me(), id)

				if r.followerLogState.getBiggerOrEqualMatchNum(newMatchIndex) > r.cluster.quorum() {
					if r.getCommitIndex() < newMatchIndex {
						r.setCommitIndex(newMatchIndex)
						r.logger.Infof("%v set current commitIndex: %d", r.me(), r.getCommitIndex())
						r.notifyCommit()
					}
				}

				r.mu.Unlock()
			}

		case <-tick:
			tick = time.After(r.heartbeatTimeout)
			r.heartbeat(heartbeatResponses)

		case <-r.shutdownCh:
			return
		}
	}
}

func (r *Raft) me() Server {
	return r.serverState.myself
}

func (r *Raft) resetElectronTimer() {
	if !r.electronTimer.Stop() && len(r.electronTimer.C) > 0 {
		<-r.electronTimer.C
	}
	r.electronTimer.Reset(RandomTimeout(r.electronTimeout))
}

type campaignReq struct {
	id   ServerID
	req  *RequestVoteRequest
	resp *RequestVoteResponse
}

func (r *Raft) campaign() <-chan *campaignReq {

	r.setCurrentTerm(r.getCurrentTerm() + 1)

	campaignRes := make(chan *campaignReq, r.cluster.size()-1)

	CandidateId := r.me().ServerID
	term := r.getCurrentTerm()
	lastIndex, lastTerm := r.getLastLog()

	f := func(s Server) {
		if s == r.me() {
			return
		}
		req := &RequestVoteRequest{
			Term:         term,
			CandidateId:  CandidateId,
			LastLogIndex: lastIndex,
			LastLogTerm:  lastTerm,
		}
		resp := &RequestVoteResponse{}
		if r.getState() == Candidate {
			err := r.trans.SendRequestVote(s, req, resp)
			if err != nil {
				r.logger.Warningf("%v fail to send RequestVote: %v", r.me(), err)
				return
			}
			campaignRes <- &campaignReq{s.ServerID, req, resp}
		}
	}

	r.cluster.visit(f, true)

	return campaignRes
}

type heartbeatRes struct {
	id   ServerID
	req  *AppendEntriesRequest
	resp *AppendEntriesResponse
}

func (r *Raft) heartbeat(c chan *heartbeatRes) {

	term := r.getCurrentTerm()
	commitIndex := r.getCommitIndex()
	leaderId := r.me().ServerID

	f := func(s Server) {
		if s == r.me() {
			return
		}

		r.mu.Lock()

		nextIndex := r.followerLogState.getNextIndex(leaderId)
		prevLogIndex, prevLogTerm := r.getPrevLog(leaderId)

		startIndex := nextIndex
		if startIndex == 0 {
			startIndex = 1
		}
		entries := r.getEntries(startIndex)

		r.mu.Unlock()

		req := &AppendEntriesRequest{
			Term:         term,
			LeaderId:     leaderId,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			LeaderCommit: commitIndex,
			Entries:      entries,
		}
		resp := &AppendEntriesResponse{}
		if r.getState() == Leader {
			err := r.trans.SendAppendEntries(s, req, resp)
			if err != nil {
				r.logger.Warningf("%v fail to send RequestVote: %v", r.me(), err)
				return
			}

			c <- &heartbeatRes{s.ServerID, req, resp}
		}
	}

	r.cluster.visit(f, true)
}

func (r *Raft) processRPC(rpc *RPC) {
	switch cmd := rpc.Command.(type) {
	case *AppendEntriesRequest:
		r.doAppendEntries(rpc, cmd)
	case *RequestVoteRequest:
		r.doRequestVote(rpc, cmd)
	default:
		panic("Unknown")
	}
}

func (r *Raft) doRequestVote(rpc *RPC, req *RequestVoteRequest) {
	defer func() {
		if r.getState() == Follower {
			r.resetElectronTimer()
		}
		rpc.Respond()
	}()

	resp := rpc.Response.(*RequestVoteResponse)
	resp.Term = r.getCurrentTerm()
	resp.VoteGranted = false

	if r.getCurrentTerm() > req.Term {
		r.logger.Infof("%v get smaller term: %d, ignoring the req: %v",
			r.me(), resp.Term, req)
		return
	}

	if r.getCurrentTerm() < req.Term {
		r.logger.Infof("%v get bigger term: %d, become Follower", r.me(), req.Term)
		r.becomeFollower()
		r.setCurrentTerm(req.Term)
		resp.Term = r.getCurrentTerm()
	}

	if r.getLeaderID() != NoneID {
		r.logger.Warningf("%v already have a leader, reject the req", r.me())
		return
	}

	if r.isVoted() {
		r.logger.Warningf("%v has voted for: %d, reject the req: %v",
			r.me(), r.getVoteFor(), req)
		return
	}

	lastIndex, lastTerm := r.getLastLog()
	if lastTerm > req.LastLogTerm {
		r.logger.Warningf("%v reject vote request, term : our(%d) > remote(%d)",
			r.me(), lastTerm, req.LastLogTerm)
		return
	}

	if lastTerm == req.LastLogTerm && lastIndex > req.LastLogIndex {
		r.logger.Warningf("%v reject vote request, commit log index: our(%d) > remote(%d)",
			r.me(), lastIndex, req.LastLogIndex)
		return
	}

	r.setVoteFor(ServerID(req.CandidateId))
	r.logger.Infof("%v vote for id: %d", r.me(), r.getVoteFor())

	resp.VoteGranted = true
}

func (r *Raft) doAppendEntries(rpc *RPC, req *AppendEntriesRequest) {
	defer func() {
		if r.serverState.getState() == Follower {
			r.resetElectronTimer()
		}
		rpc.Respond()
	}()

	resp := rpc.Response.(*AppendEntriesResponse)
	resp.Term = r.serverState.getCurrentTerm()
	resp.Success = false

	if r.getCurrentTerm() > req.Term {
		r.logger.Infof("%v get smaller term: %d, ignore the req: %v",
			r.me(), req.Term, req)
		return
	}

	if r.getCurrentTerm() < req.Term || r.getState() != Follower {
		r.logger.Infof("%v get bigger term: %d or receive error req: %v, become Follower",
			r.me(), req.Term, req)
		r.becomeFollower()
		r.setCurrentTerm(req.Term)
		resp.Term = r.getCurrentTerm()
	}

	r.setLeaderID(ServerID(req.LeaderId))

	if req.PrevLogIndex > 0 {
		lastIndex, lastTerm := r.getLastLog()

		var prevLogTerm uint64
		if lastIndex == req.PrevLogIndex {
			prevLogTerm = lastTerm
		} else {
			prevLog, err := r.logs.GetEntry(req.PrevLogIndex)
			if err != nil {
				// follower does not have PrevLogIndex in its log
				resp.ConflictIndex = lastIndex

				r.logger.Infof("%v failed to get previous log previous-index: %v last-index: %v error: %v",
					r.me(), req.PrevLogIndex, lastIndex, err)
				return
			}
			prevLogTerm = prevLog.Term
		}

		if prevLogTerm != req.PrevLogTerm {
			r.logger.Infof("%v previous log term mis-match ours: %v remote: %v id: %v",
				r.me(), prevLogTerm, req.PrevLogTerm, req.LeaderId)

			prevLog, _ := r.logs.GetEntry(req.PrevLogIndex)

			isFindEqualTerm := false
			for index := prevLog.Index; index > 0; index-- {
				entry, err := r.logs.GetEntry(index)
				if err != nil {
					panic(err)
				}

				if entry.Term == prevLog.Term {
					isFindEqualTerm = true
					resp.ConflictIndex = index
				} else {
					if isFindEqualTerm {
						break
					}
				}
			}
			return
		}
	}

	if len(req.Entries) > 0 {
		lastIndex, _ := r.getLastLog()
		var newEntries []*Entry
		for i, entry := range req.Entries {
			if entry.Index > lastIndex {
				for j := i; j < len(req.Entries); j++ {
					newEntries = append(newEntries, &req.Entries[j])
				}
				break
			}

			storeEntry, err := r.logs.GetEntry(entry.Index)
			if err != nil {
				r.logger.Infof("%v failed to get log entry index: %v error: %v",
					r.me(), entry.Index, err)
				return
			}

			if entry.Term != storeEntry.Term {
				r.logger.Infof("%v clearing log suffix from: %v to: %v",
					r.me(), entry.Index, lastIndex)
				if err := r.logs.DeleteRange(entry.Index, lastIndex); err != nil {
					panic(err)
				}
				for j := i; j < len(req.Entries); j++ {
					newEntries = append(newEntries, &req.Entries[j])
				}
				break
			}
		}

		if n := len(newEntries); n > 0 {
			r.logger.Infof("%v store entries: %#v", r.me(), len(newEntries))
			err := r.logs.StoreEntries(newEntries)
			if err != nil {
				panic(err)
			}
		}
	}

	commitIndex := r.getCommitIndex()
	if req.LeaderCommit > 0 && req.LeaderCommit > commitIndex {
		index, err := r.logs.LastIndex()
		if err != nil {
			panic(err)
		}
		idx := Min(req.LeaderCommit, index)
		r.setCommitIndex(idx)
		r.logger.Infof("%v set current commit index to: %d", r.me(), idx)

		r.notifyCommit()
	}

	resp.Success = true
}
