package main

import (
	"fmt"
	"sync"

	v1alpha1 "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

type votes struct {
	l sync.RWMutex

	hashes     map[[32]byte]uint
	roots      map[[32]byte]uint
	counts     map[uint64]uint
	votes      map[[32]byte]*v1alpha1.Eth1Data
	voteCounts map[[32]byte]uint
	total      uint
}

func newVotes() *votes {
	return &votes{
		hashes:     make(map[[32]byte]uint),
		roots:      make(map[[32]byte]uint),
		counts:     make(map[uint64]uint),
		votes:      make(map[[32]byte]*v1alpha1.Eth1Data),
		voteCounts: make(map[[32]byte]uint),
	}
}

func (v *votes) Report() string {
	v.l.RLock()
	defer v.l.RUnlock()
	format := `====Eth1Data Voting Report====

Total votes: %d

Block Hashes
%s
Deposit Roots
%s
Deposit Counts
%s
Votes
%s
`
	var blockHashes string
	for r, cnt := range v.hashes {
		blockHashes += fmt.Sprintf("%#x=%d\n", r, cnt)
	}
	var depositRoots string
	for r, cnt := range v.roots {
		depositRoots += fmt.Sprintf("%#x=%d\n", r, cnt)
	}
	var depositCounts string
	for dc, cnt := range v.counts {
		depositCounts += fmt.Sprintf("%d=%d\n", dc, cnt)
	}
	var votes string
	for htr, e1d := range v.votes {
		votes += fmt.Sprintf("%s=%d\n", e1d.String(), v.voteCounts[htr])
	}

	return fmt.Sprintf(
		format,
		v.total,
		blockHashes,
		depositRoots,
		depositCounts,
		votes,
	)
}
