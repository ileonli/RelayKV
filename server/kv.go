package main

import (
	"RelayKV/raft"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type Method string

const (
	GET Method = "GET"
	PUT        = "PUT"
)

type Pair struct {
	X, Y interface{}
}

type Op struct {
	Method Method // 本次操作类型，分为：GET, PUT, DELETE
	Key    string // 本次操作键值对的键
	Value  string // 本次操作键值对的值
	Clerk  string // 发出请求的客户端唯一标识
	Index  uint64 // 客户端产生操作的索引，递增
	Error  string // 用于保存处理过程的错误信息
}

type KVServer struct {
	mu sync.Mutex

	rf    *raft.Raft
	clerk string
	index uint64

	httpAddress []string

	kv map[string]string

	clerkLog map[string]uint64
	msgCh    map[Pair]chan *Op
}

func NewKVService(rf *raft.Raft, clerk string, httpAddress []string) *KVServer {
	return &KVServer{
		rf:          rf,
		clerk:       clerk,
		httpAddress: httpAddress,
		kv:          make(map[string]string),
		clerkLog:    make(map[string]uint64),
		msgCh:       make(map[Pair]chan *Op),
	}
}

func (s *KVServer) IncrIndex() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.index++
	return s.index
}

type Response struct {
	Code    int
	Key     string
	Value   string
	Message string
}

func (s *KVServer) Get(key string) Response {
	op := Op{GET, key, "", s.clerk, s.IncrIndex(), ""}
	_, _, isLeader := s.rf.Start(op)
	pair := Pair{op.Clerk, op.Index}
	if isLeader {
		s.mu.Lock()
		s.msgCh[pair] = make(chan *Op)
		s.mu.Unlock()

		defer func() {
			close(s.msgCh[pair])
			delete(s.msgCh, pair)
		}()
	} else {
		leaderId := s.rf.GetLeaderId()
		httpAddress := s.httpAddress[leaderId]
		resp, err := http.Get("http://" + httpAddress + "/GET?KEY=" + key)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		var res Response
		bytes, err := ioutil.ReadAll(resp.Body)
		err = json.Unmarshal(bytes, &res)
		if err != nil {
			panic(err)
		}
		if res.Value == DELETE {
			res.Value = ""
		}
		return res
	}

	select {
	case opRes := <-s.msgCh[pair]:
		return Response{Code: 200, Key: key, Value: opRes.Value}
	case <-time.After(time.Duration(3) * time.Second):
		return Response{Code: 5001, Message: "超时"}
	}
}

func (s *KVServer) Put(key, val string) Response {
	op := Op{PUT, key, val, s.clerk, s.IncrIndex(), ""}
	_, _, isLeader := s.rf.Start(op)
	pair := Pair{op.Clerk, op.Index}
	if isLeader {
		s.mu.Lock()
		s.msgCh[pair] = make(chan *Op)
		s.mu.Unlock()

		defer func() {
			close(s.msgCh[pair])
			delete(s.msgCh, pair)
		}()

	} else {
		leaderId := s.rf.GetLeaderId()
		httpAddress := s.httpAddress[leaderId]
		resp, err := http.Get("http://" + httpAddress + "/PUT?KEY=" + key + "&VALUE=" + val)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		var res Response
		bytes, err := ioutil.ReadAll(resp.Body)

		err = json.Unmarshal(bytes, &res)
		if err != nil {
			panic(err)
		}
		return res
	}
	select {
	case opRes := <-s.msgCh[pair]:
		return Response{Code: 200, Key: key, Value: opRes.Value}
	case <-time.After(time.Duration(3) * time.Second):
		return Response{Code: 5001, Message: "超时"}
	}
}

const DELETE = "DELETED_SYMBOL"

func (s *KVServer) Delete(key string) Response {
	return s.Put(key, DELETE)
}

func (s *KVServer) ApplyOpToStorage(op *Op) *Op {
	index, ok := s.clerkLog[op.Clerk]
	if ok && index >= op.Index && op.Method == PUT {
		op.Error = "已经执行过了"
	} else {
		s.clerkLog[op.Clerk] = Max(s.clerkLog[op.Clerk], op.Index)
		switch op.Method {
		case GET:
			if s.kv[op.Key] != DELETE {
				op.Value = s.kv[op.Key]
			}
		case PUT:
			s.kv[op.Key] = op.Value
		default:
			op.Error = "无此操作"
		}
		if ch, ok := s.msgCh[Pair{op.Clerk, op.Index}]; ok {
			ch <- op
		}
	}
	return op
}

type Status struct {
	State string
	Term  uint64
	Log   map[string]string
}

func (s *KVServer) Status() Status {
	log := make(map[string]string)
	for k, v := range s.kv {
		if v != DELETE {
			log[k] = v
		}
	}
	return Status{
		State: s.rf.GetState().String(),
		Term:  s.rf.GetTerm(),
		Log:   log,
	}
}
