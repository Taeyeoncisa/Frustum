package Node

import "BlockChain/ID"

type State struct {
	IntraTerm  uint
	GlobalTerm uint

	IntraRole     string
	GlobalRole    string
	BlockHeight   uint64
	ConsensusType string

	IntraCandiStat *CandiDateState
	IntraLeader    ID.NodeID

	IntraQ1Count *Counter
	IntraQ2Count *BlockCount
	/*************************/
	//一个ShardLeader连续挖矿的次数，该计数器抵达上限后切换下一个ShardLeader
	FRaftBatchBlockCount int
}

type CandiDateState struct {
	Id   ID.NodeID
	Term uint
}

func NewCandiDateState() *CandiDateState {
	value := new(CandiDateState)
	value.Id = ID.NodeID{}
	value.Term = 0
	return value
}

type Counter struct {
	Count uint
	Term  uint
}

type BlockCount struct {
	Count            uint
	GlobalCount      uint
	LatestResponseID ID.NodeID
	BlockHeight      uint64
}

func NewCounter() *Counter {
	value := new(Counter)
	value.Count = 0
	value.Term = 0
	return value
}

func NewBlockCounter() *BlockCount {
	value := new(BlockCount)
	value.Count = 0
	value.GlobalCount = 0
	value.BlockHeight = 0
	value.LatestResponseID = ID.NodeID{}
	return value
}

func NewState(consensus string) *State {

	value := State{
		IntraTerm:      0,
		GlobalTerm:     0,
		IntraRole:      "intraFollower",
		GlobalRole:     "",
		BlockHeight:    0,
		ConsensusType:  consensus,
		IntraCandiStat: NewCandiDateState(),
		IntraLeader:    ID.NodeID{},
		IntraQ1Count:   NewCounter(),
		IntraQ2Count:   NewBlockCounter(),
	}
	return &value
}
