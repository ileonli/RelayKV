package raft

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func Min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func Max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

// RandomTimeout returns a value that is between the minVal and 2x minVal.
func RandomTimeout(minVal time.Duration) time.Duration {
	if minVal == 0 {
		return 0
	}
	extra := time.Duration(rand.Int63()) % minVal
	return minVal + extra
}
