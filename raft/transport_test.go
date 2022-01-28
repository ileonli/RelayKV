package raft

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"testing"
)

func TestNewHttpTransport(t *testing.T) {
	NewHttpTransport(":8610", NewLogrusLogger(os.Stdout, logrus.WarnLevel))

	select {}
}

func Test(t *testing.T) {
	t1 := NewHttpTransport(":8611", NewLogrusLogger(os.Stdout, logrus.WarnLevel))
	req := &RequestVoteRequest{
		Term: 100,
	}
	resp := &RequestVoteResponse{}
	t1.SendRequestVote(Server{
		ServerID:      0,
		ServerAddress: "127.0.0.1:8610",
	}, req, resp)

	fmt.Printf("%#v", *resp)
}
