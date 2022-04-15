package main

import (
	"RelayKV/raft"
	"encoding/json"
	"flag"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

func parseClusterPath(rawClusterPath string) []string {
	return strings.Split(rawClusterPath, ",")
}

func NewRaftService(nodePaths []string, nodeId int,
	apply chan *raft.ApplyMsg, logger raft.Logger) *raft.Raft {

	cluster := raft.NewCluster(nodePaths)

	configuration := raft.NewConfiguration(cluster[nodeId],
		cluster, time.Millisecond*1000, time.Millisecond*300)

	trans := raft.NewRPCTransport(raft.ServerAddress(nodePaths[nodeId]), logger)

	logStore := raft.NewInMemoryEntryStore()

	persistStore := raft.NewInMemoryPersistStore()

	return raft.NewRaft(apply, configuration, logger, trans, logStore, persistStore)
}

func main() {
	clusterHTTP := flag.String("http", "127.0.0.1:9870,127.0.0.1:9871,127.0.0.1:9872", "http address")
	clusterRPC := flag.String("cluster", "127.0.0.1:7651,127.0.0.1:7652,127.0.0.1:7653", "cluster address")
	nodeId := flag.Int("id", 0, "current node id")
	flag.Parse()

	httpAddress := parseClusterPath(*clusterHTTP)
	rpcAddress := parseClusterPath(*clusterRPC)

	logger := raft.NewLogrusLogger(os.Stdout, logrus.TraceLevel)

	apply := make(chan *raft.ApplyMsg, 1)

	rf := NewRaftService(rpcAddress, *nodeId, apply, logger)

	kvService := NewKVService(rf, httpAddress[*nodeId], httpAddress)

	go func() {
		for {
			select {
			case msg := <-apply:
				var op Op
				err := json.Unmarshal(msg.Data, &op)
				if err != nil {
					panic(err)
				}
				kvService.ApplyOpToStorage(&op)
			}
		}
	}()

	NewHttpService(httpAddress[*nodeId], kvService)
}
