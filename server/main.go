package main

import (
	"RelayKV/raft"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {

	var cluster raft.Cluster

	args := os.Args[1:]

	leaderId, _ := strconv.Atoi(args[0])
	clusterPath := args[1]
	nodes := strings.Split(clusterPath, ",")
	for _, node := range nodes {
		idAndAddress := strings.Split(node, "=")
		id, err := strconv.Atoi(idAndAddress[0])
		if err != nil {
			panic(err)
		}
		address := idAndAddress[1]
		cluster = append(cluster, raft.Server{
			ServerID:      raft.ServerID(id),
			ServerAddress: raft.ServerAddress(address),
		})
	}

	configuration := raft.NewConfiguration(cluster[leaderId],
		cluster, time.Millisecond*1000, time.Millisecond*50)

	applyCh := make(chan *raft.ApplyMsg, 1)
	logger := raft.NewLogrusLogger(os.Stdout, logrus.TraceLevel)
	address := ":" + raft.ServerAddress(strings.Split(string(cluster[leaderId].ServerAddress), ":")[1])
	trans := raft.NewRPCTransport(address, logger)
	logStore := raft.NewInMemoryEntryStore()
	persistStore := raft.NewInMemoryPersistStore()

	raft.NewRaft(applyCh, configuration, logger, trans, logStore, persistStore)

	select {}
}
