package raft

import (
	"fmt"
)

type ServerID uint64

type ServerAddress string

const (
	NoneID      ServerID      = 0
	NoneAddress ServerAddress = ""
)

type Server struct {
	ServerID
	ServerAddress
}

func (s Server) String() string {
	return fmt.Sprintf("Server{ ID: %v, Address: %v }",
		s.ServerID, s.ServerAddress)
}

type Cluster []Server

func (c *Cluster) size() uint64 {
	return uint64(len(*c))
}

func (c *Cluster) quorum() uint64 {
	return uint64(len(*c))
}

func (c *Cluster) at(index uint64) Server {
	if index > c.size() {
		panic("index out of bound of the cluster")
	}
	return (*c)[index]
}

func (c *Cluster) visit(f func(s Server)) {
	for i := uint64(0); i < c.size(); i++ {
		f(c.at(i))
	}
}
