package NetWork

import (
	"BlockChain/ID"
)

const IntraLeaderShardID = 1024
const GlobalLeaderShardID = 1025

type Shard struct {
	Member map[string]string
}

func NewShard() *Shard {

	shard := new(Shard)
	shard.Member = make(map[string]string)
	return shard

}

func (s *Shard) AddNode(id, ipAddr string) (bool, string) {

	if url, ok := s.Member[id]; ok {
		return false, url
	} else {
		s.Member[id] = ipAddr
		return true, ipAddr
	}

}

func (s *Shard) FindNode(id string) (bool, string) {

	if url, ok := s.Member[id]; ok {
		return true, url
	} else {
		return false, "NodeNotExist"
	}

}

func (s *Shard) Length() int {
	return len(s.Member)
}

type NodeTable struct {
	SelfNodeID string
	Shards     map[int]*Shard
}

func NewNodeTable(id string) *NodeTable {

	table := new(NodeTable)
	table.SelfNodeID = id
	table.Shards = make(map[int]*Shard)

	table.Shards[IntraLeaderShardID] = NewShard()
	table.Shards[GlobalLeaderShardID] = NewShard()
	table.Shards[ID.StringToID(table.SelfNodeID).ShardID] = NewShard()

	return table

}

func (nt *NodeTable) AddNode(id, ipAddr string) (bool, string) {

	nodeId := ID.StringToID(id)

	if shard, shardExit := nt.Shards[nodeId.ShardID]; shardExit {

		return shard.AddNode(id, ipAddr)

	} else {

		nt.Shards[nodeId.ShardID] = NewShard()

		return nt.Shards[nodeId.ShardID].AddNode(id, ipAddr)

	}

}

func (nt *NodeTable) FindNode(id string) (bool, string) {

	nodeId := ID.StringToID(id)

	if shard, shardExit := nt.Shards[nodeId.ShardID]; shardExit {

		return shard.FindNode(id)

	} else {

		return nt.Shards[1024].FindNode(id)

	}

}

// FindMyIntraLeader 返回值bool代表是否找到，第一个string为leaderID，第二个string为leader地址
func (nt *NodeTable) FindMyIntraLeader(id string) (bool, string, string) {

	intraLeaderShard := nt.Shards[IntraLeaderShardID]
	shardID := ID.StringToID(id).ShardID

	for k, v := range intraLeaderShard.Member {

		if ID.StringToID(k).ShardID == shardID {
			return true, k, v
		}

	}

	return false, "IntraLeaderNotExist", "unknownURL"

}

// FindMyGlobalLeader 返回值bool代表是否找到，第一个string为leaderID，第二个string为leader地址
func (nt *NodeTable) FindMyGlobalLeader() (bool, string, string) {

	globalLeaderShard := nt.Shards[GlobalLeaderShardID]
	var globalLeader, url string

	if len(globalLeaderShard.Member) == 1 {

		for k, v := range globalLeaderShard.Member {
			globalLeader = k
			url = v
		}
		return true, globalLeader, url

	} else {
		return false, "GlobalLeaderNotExist", "unknownURL"
	}

}

// HandleShardMsg 返回值1为分片ID，返回值2为本节点是否为leader
func (nt *NodeTable) HandleShardMsg(msgType string, shard *map[string]string) (int, bool) {

	var shardID int
	var ImLeader bool

	switch msgType {
	case "IntraLeaderMem":
		shardID = 1024

		for k := range *shard {
			if k == nt.SelfNodeID {
				ImLeader = true
			}
		}

	case "GlobalLeaderMem":
		shardID = 1025

		for k := range *shard {
			if k == nt.SelfNodeID {
				ImLeader = true
			}
		}
	default:
		for k := range *shard {
			shardID = ID.StringToID(k).ShardID
			break
		}
	}
	nt.Shards[shardID] = &Shard{Member: *shard}

	return shardID, ImLeader
}

func (nt *NodeTable) FindNodeMsg() *map[int]int {

	newMap := make(map[int]int)
	for k, v := range nt.Shards {
		newMap[k] = len(v.Member)
	}

	return &newMap
}

func (nt *NodeTable) HandleFindNodeMsg(findNode *map[int]int) *map[int]*map[string]string {

	newMap := make(map[int]*map[string]string)

	for k, v := range *findNode {

		if v == len(nt.Shards[k].Member) {
			//maybe has the same shard member
		} else {
			shard := nt.Shards[k].Member
			newMap[k] = &shard
		}

	}

	return &newMap
}
