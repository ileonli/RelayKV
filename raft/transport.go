package raft

import (
	"RelayKV/raft/pb"
	"context"
	"github.com/shimingyah/pool"
	"google.golang.org/grpc"
	"net"
	"sync"
)

type RPC struct {
	Command  interface{}
	Response interface{}
	waitCh   chan struct{}
}

func (r *RPC) Respond() {
	r.waitCh <- struct{}{}
}

type Transport interface {
	Consume() <-chan *RPC

	SendRequestVote(server Server, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error)

	SendAppendEntries(server Server, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error)
}

type RPCTransport struct {
	pb.UnimplementedRaftServer

	clientPool sync.Map
	rpcCh      chan *RPC
	logger     Logger
}

func NewRPCTransport(address ServerAddress, logger Logger) *RPCTransport {
	transport := &RPCTransport{
		rpcCh:  make(chan *RPC, 1),
		logger: logger,
	}

	go func() {
		server, err := net.Listen("tcp", string(address))
		if err != nil {
			panic(err)
		}
		var rpcServer = grpc.NewServer()
		pb.RegisterRaftServer(rpcServer, transport)

		err = rpcServer.Serve(server)
		if err != nil {
			panic(err)
		}
	}()
	return transport
}

func (t *RPCTransport) Consume() <-chan *RPC {
	return t.rpcCh
}

func (t *RPCTransport) SendRequestVote(
	server Server, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {

	clientPool, ok := t.clientPool.Load(string(server.ServerAddress))
	if !ok {
		p, err := pool.New(string(server.ServerAddress), pool.DefaultOptions)
		if err != nil {
			t.logger.Fatalf("failed to new pool: %v", err)
		}
		t.clientPool.Store(string(server.ServerAddress), p)
		clientPool = p
	}
	conn, err := clientPool.(pool.Pool).Get()
	if err != nil {
		t.logger.Fatalf("failed to get conn: %v", err)
	}
	defer conn.Close()

	client := pb.NewRaftClient(conn.Value())
	response, err := client.RequestVote(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return response, err
}

func (t *RPCTransport) SendAppendEntries(
	server Server, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {

	clientPool, ok := t.clientPool.Load(string(server.ServerAddress))
	if !ok {
		p, err := pool.New(string(server.ServerAddress), pool.DefaultOptions)
		if err != nil {
			t.logger.Fatalf("failed to new pool: %v", err)
		}
		t.clientPool.Store(string(server.ServerAddress), p)
		clientPool = p
	}
	conn, err := clientPool.(pool.Pool).Get()
	if err != nil {
		t.logger.Fatalf("failed to get conn: %v", err)
	}
	defer conn.Close()

	client := pb.NewRaftClient(conn.Value())
	response, err := client.AppendEntries(context.Background(), req)
	if err != nil {
		t.logger.Warning(err)
	}
	return response, err
}

func (t *RPCTransport) waitResponse(req interface{}, resp interface{}) {

	waitCh := make(chan struct{}, 1)
	r := &RPC{
		Command:  req,
		Response: resp,
		waitCh:   waitCh,
	}
	t.rpcCh <- r

	<-waitCh
}

func (t *RPCTransport) RequestVote(ctx context.Context, request *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	resp := &pb.RequestVoteResponse{}
	t.waitResponse(request, resp)
	return resp, nil
}

func (t *RPCTransport) AppendEntries(ctx context.Context, request *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	resp := &pb.AppendEntriesResponse{}
	t.waitResponse(request, resp)
	return resp, nil
}
