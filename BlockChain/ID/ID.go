package ID

import (
	"fmt"
	"strconv"
	"strings"
)

// NodeID 分片ID和片内ID
type NodeID struct {
	ShardID   int
	InShardID int
}

// 返回string的节点ID，格式为“ShardID-InShardID”
func (ni *NodeID) String() string {

	shardId := strconv.Itoa(ni.ShardID)
	inShardId := strconv.Itoa(ni.InShardID)

	return fmt.Sprintf("%s-%s", shardId, inShardId)

}

func StringToID(str string) NodeID {

	idSlice := strings.Split(str, "-")
	id := NodeID{
		ShardID:   0,
		InShardID: 0,
	}

	id.ShardID, _ = strconv.Atoi(idSlice[0])
	id.InShardID, _ = strconv.Atoi(idSlice[1])
	return id
}
