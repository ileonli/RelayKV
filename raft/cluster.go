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

type Cluster []Server

type Server struct {
	ServerID
	ServerAddress
}

func (s Server) String() string {
	return fmt.Sprintf("Server{ ID: %v, Address: %v }",
		s.ServerID, s.ServerAddress)
}

func (s *Server) getServerID() ServerID {
	return s.ServerID
}

func (s *Server) getServerAddress() ServerAddress {
	return s.ServerAddress
}
