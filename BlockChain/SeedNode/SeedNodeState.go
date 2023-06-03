package SeedNode

type State struct {
	NodeRole      string
	ConsensusType string
	ConsensusStop int
	//Shards *NetWork.ShardManager
}

func NewState() *State {

	state := State{
		NodeRole:      "SeedNode",
		ConsensusType: "",
		ConsensusStop: 0,
	}

	return &state
}
