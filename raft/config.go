package raft

import "time"

type Configuration struct {
	Me Server

	Cluster Cluster

	ElectronTimeout, HeartbeatTimeout time.Duration
}

func NewConfiguration(me Server, c Cluster,
	electronTimeout, heartbeatTimeout time.Duration) *Configuration {
	return &Configuration{
		Me:               me,
		Cluster:          c,
		ElectronTimeout:  electronTimeout,
		HeartbeatTimeout: heartbeatTimeout,
	}
}

// TODO finish verifyConfig
func verifyConfig(c *Configuration) {

}
