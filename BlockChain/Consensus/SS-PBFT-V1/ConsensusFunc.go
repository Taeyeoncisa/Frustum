package SS_PBFT_V1

import (
	"BlockChain/DB"
	"BlockChain/ID"
	"BlockChain/Message"
	"time"
)

const QuorumPercentage = 0.5
const StopMiningBlockHeight = 64

// TrNumInBlock 区块中包含交易数量
const TrNumInBlock = 1024

func (n *Node) StartConsensus() {
	n.NodeState.LatestBlock = n.GenerateNewBlock()
	newBlock := *n.NodeState.LatestBlock
	newBlock.BlockBody = nil

	n.ResetConsensusCheckCount()

	msg := Message.NewNodeMsg(n.NodeNet.SelfNodeID, n.NodeNet.SelfNodeID,
		"PrePrepare", newBlock)
	msg.EpochNum = n.NodeState.NodeEpoch
	msg.BlockHeight = n.NodeState.BlockHeight
	n.BroadCastV2("shard", n.NodeNet.SelfNodeID.ShardID, msg)

	msgWithBlock := Message.NewNodeMsg(n.NodeNet.SelfNodeID, n.NodeNet.SelfNodeID,
		"block", n.NodeState.LatestBlock)
	n.BroadCastV2("shard", 1024, msgWithBlock)

	//直接同步发送Prepare可能存在其他节点无法触发commit的问题
	//所以协程启动+延迟5ms
	go n.SendPrepare()
	go n.InformSeedNodeMiningStat()
	go n.NodeNet.LoadStats.CalculateTPS(n.NodeState.LatestBlock)

}

func (n *Node) SendPrepare() {
	msg := Message.NewNodeMsg(n.NodeNet.SelfNodeID, n.NodeNet.SelfNodeID,
		"Prepare", nil)
	msg.EpochNum = n.NodeState.NodeEpoch
	msg.BlockHeight = n.NodeState.BlockHeight
	n.BroadCast("shard", n.NodeNet.SelfNodeID.ShardID, msg)
}

func (n *Node) SendCommit() {
	msg := Message.NewNodeMsg(n.NodeNet.SelfNodeID, n.NodeNet.SelfNodeID,
		"Commit", nil)
	msg.EpochNum = n.NodeState.NodeEpoch
	msg.BlockHeight = n.NodeState.BlockHeight
	n.BroadCast("shard", n.NodeNet.SelfNodeID.ShardID, msg)
}

func (n *Node) SendFinalToIntraLeader() {
	msg := Message.NewNodeMsg(n.NodeNet.SelfNodeID, n.NodeNet.SelfNodeID,
		"FinalCommit", nil)
	msg.EpochNum = n.NodeState.NodeEpoch
	msg.BlockHeight = n.NodeState.BlockHeight

	id, url := n.NodeNet.Shards.FindMyIntraLeader(n.NodeNet.SelfNodeID)
	msg.To = id
	n.NodeNet.SendNodeMsg(url, msg)

	if n.NodeState.BlockHeight == StopMiningBlockHeight {
		n.NodeNet.LoadStats.PrintResult()
	}
}

// GenerateNewBlock 生成新区块，更新节点的区块高度信息
func (n *Node) GenerateNewBlock() *DB.Block {
	newBlock := new(DB.Block)

	newBlock.BlockBody = *n.NodeTxPool.CommitTrs(TrNumInBlock)
	//更新节点记录的最新区块高度
	n.NodeState.BlockHeight++
	newBlock.BlockHeight = n.NodeState.BlockHeight

	newBlock.TimeStamp = time.Now().UnixNano()
	newBlock.Miner = n.NodeNet.SelfNodeID
	newBlock.PreBlockHash = n.NodeState.PreBlockHash

	newBlock.SetHash()
	//更新节点记录的最新区块哈希
	n.NodeState.PreBlockHash = newBlock.BlockHash

	return newBlock
}

// ResetConsensusCheckCount 该函数重置共识计票器，更新最新的区块高度
func (n *Node) ResetConsensusCheckCount() {
	//后续判定共识消息是否属于本轮出块

	n.NodeState.PrepareCheck.ResetCount(n.NodeState.BlockHeight)
	n.NodeState.CommitCheck.ResetCount(n.NodeState.BlockHeight)
	n.NodeState.FinalCommitCheck.ResetCount(n.NodeState.BlockHeight)

}

func (n *Node) InformSeedNodeMiningStat() {
	seedNode := ID.NodeID{
		ShardID:   0,
		InShardID: 0,
	}
	msg := Message.NewNodeMsg(n.NodeNet.SelfNodeID, seedNode, "MiningStat", nil)
	msg.EpochNum = n.NodeState.NodeEpoch
	msg.BlockHeight = n.NodeState.BlockHeight

	n.NodeCh.SendMsgCh <- msg
}

func (n *Node) InformSeedNodeMiningStop() {
	seedNode := ID.NodeID{
		ShardID:   0,
		InShardID: 0,
	}
	msg := Message.NewNodeMsg(n.NodeNet.SelfNodeID, seedNode, "MiningStop", nil)
	msg.EpochNum = n.NodeState.NodeEpoch
	msg.BlockHeight = n.NodeState.BlockHeight

	n.NodeCh.SendMsgCh <- msg
}
