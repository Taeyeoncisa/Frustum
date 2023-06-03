package SS_PBFT_V1

import (
	"BlockChain/Consensus"
	"BlockChain/DB"
)

type State struct {
	NodeRole  string
	NodeEpoch int

	/**************************/

	BlockHeight  uint64
	PreBlockHash []byte
	LatestBlock  *DB.Block

	/**************************/

	PrepareCheck     *Consensus.Quorum
	CommitCheck      *Consensus.Quorum
	FinalCommitCheck *Consensus.Quorum

	/**************************/

}

func NewState() *State {

	return &State{
		NodeRole:         "follower",
		NodeEpoch:        0,
		BlockHeight:      0,
		PreBlockHash:     nil,
		PrepareCheck:     Consensus.NewQuorum(1, Consensus.IntraScope),
		CommitCheck:      Consensus.NewQuorum(1, Consensus.IntraScope),
		FinalCommitCheck: Consensus.NewQuorum(1, Consensus.IntraScope),
	}

}
