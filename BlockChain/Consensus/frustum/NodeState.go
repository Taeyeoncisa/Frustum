package SS_PBFT_V2

import (
	"BlockChain/Consensus/PBFT"
	"BlockChain/DB"
	"BlockChain/ID"
	"sync"
)

type State struct {
	NodeID ID.NodeID
	Role   string //IntraLeader, GlobalLeader, follower
	Stage  string //initialize, prepare, commit, finality

	BlockHeight  uint64
	PreBlockHash []byte
	LatestBlock  *DB.Block

	/******************************************************/

	PrepareCheck  map[string]*PBFT.Quorum
	CommitCheck   map[string]*PBFT.Quorum
	FinalityCheck map[string]*PBFT.Quorum

	/******************************************************/
	Flag sync.Mutex
}

func NewState(nodeId string) *State {

	prepareCheck := make(map[string]*PBFT.Quorum)
	prepareCheck["IntraPrepare"] = PBFT.NewQuorum("IntraPrepare")
	prepareCheck["GlobalPrepare"] = PBFT.NewQuorum("GlobalPrepare")

	commitCheck := make(map[string]*PBFT.Quorum)
	commitCheck["IntraCommit"] = PBFT.NewQuorum("IntraCommit")
	commitCheck["GlobalCommit"] = PBFT.NewQuorum("GlobalCommit")

	finalityCheck := make(map[string]*PBFT.Quorum)
	finalityCheck["IntraFinality"] = PBFT.NewQuorum("IntraFinality")
	finalityCheck["GlobalFinality"] = PBFT.NewQuorum("GlobalFinality")

	return &State{
		NodeID:        ID.StringToID(nodeId),
		Role:          "follower",
		Stage:         "initialize",
		BlockHeight:   0,
		PreBlockHash:  nil,
		LatestBlock:   nil,
		PrepareCheck:  prepareCheck,
		CommitCheck:   commitCheck,
		FinalityCheck: finalityCheck,
	}

}
