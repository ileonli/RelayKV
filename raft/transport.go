package raft

import (
	"log"
	"net"
	"net/rpc"
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

	SendRequestVote(server Server, req *RequestVoteRequest, resp *RequestVoteResponse) error

	SendAppendEntries(server Server, req *AppendEntriesRequest, resp *AppendEntriesResponse) error
}

type RPCTransport struct {
	rpcCh  chan *RPC
	logger Logger
}

func NewHttpTransport(address ServerAddress, logger Logger) *RPCTransport {
	transport := &RPCTransport{rpcCh: make(chan *RPC, 1), logger: logger}
	go func() {
		err := rpc.Register(transport)
		if err != nil {
			panic(err)
		}
		listener, err := net.Listen("tcp", string(address))
		if err != nil {
			log.Fatal("ListenTCP error:", err)
		}

		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("Accept error:", err)
		}
		rpc.ServeConn(conn)
	}()
	return transport
}

func (t *RPCTransport) Consume() <-chan *RPC {
	return t.rpcCh
}

func (t *RPCTransport) SendRequestVote(
	server Server, req *RequestVoteRequest, resp *RequestVoteResponse) error {

	client, err := rpc.Dial("tcp", string(server.getServerAddress()))
	if err != nil {
		return err
	}

	err = client.Call("RPCTransport.RequestVote", req, resp)
	if err != nil {
		return err
	}
	return nil
}

func (t *RPCTransport) SendAppendEntries(
	server Server, req *AppendEntriesRequest, resp *AppendEntriesResponse) error {

	client, err := rpc.Dial("tcp", string(server.getServerAddress()))
	if err != nil {
		return err
	}

	err = client.Call("RPCTransport.AppendEntries", req, resp)
	if err != nil {
		return err
	}
	return nil
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

func (t *RPCTransport) RequestVote(req *RequestVoteRequest, resp *RequestVoteResponse) error {
	t.waitResponse(req, resp)
	return nil
}

func (t *RPCTransport) AppendEntries(req *AppendEntriesRequest, resp *AppendEntriesResponse) error {
	t.waitResponse(req, resp)
	return nil
}
