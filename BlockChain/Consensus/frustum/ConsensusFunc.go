package SS_PBFT_V2

import (
	"BlockChain/DB"
	"BlockChain/ID"
	"BlockChain/Message"
	"time"
)

// TrNumInBlock 区块中包含交易数量
const TrNumInBlock = 1024

const EndBlockHeight = 64

func (n *Node) GenerateNewBlock() *DB.Block {

	newBlock := new(DB.Block)

	newBlock.BlockBody = *n.NodeTxPool.CommitTrs(TrNumInBlock)
	n.NodeState.BlockHeight++
	newBlock.BlockHeight = n.NodeState.BlockHeight

	newBlock.Miner = n.NetWork.ServerID
	newBlock.PreBlockHash = n.NodeState.PreBlockHash

	newBlock.TimeStamp = time.Now().UnixNano()
	newBlock.SetHash()

	n.NodeState.PreBlockHash = newBlock.BlockHash

	return newBlock
}

func (n *Node) ResetVoteCheck(newBlockHeight uint64) {

	//prepareCheck分为片内和片外
	for _, v := range n.NodeState.PrepareCheck {
		v.ResetCount(newBlockHeight)
	}
	for _, v1 := range n.NodeState.CommitCheck {
		v1.ResetCount(newBlockHeight)
	}
	for _, v2 := range n.NodeState.FinalityCheck {
		v2.ResetCount(newBlockHeight)
	}

}

func (n *Node) StartConsensus() {

	n.NodeState.Stage = "prepare"
	//生成新区块
	n.NodeState.LatestBlock = n.GenerateNewBlock()
	//重置共识过程中的计票器
	n.ResetVoteCheck(n.NodeState.BlockHeight)

	//片内片外发送pre-prepare消息，启动共识
	msg := Message.NewNodeMsg(n.NetWork.ServerID, "", "PrePrepare", nil)
	msg.BlockHeight = n.NodeState.BlockHeight
	n.BroadCastWithInShard(n.NodeState.NodeID.ShardID, msg)
	n.BroadCastWithInShard(1024, msg)

	//发送prepare消息
	go func() {
		//time.Sleep(1 * time.Second)
		n.SendPrepare(n.NodeState.NodeID.ShardID)
		n.SendPrepare(1024)
		n.DialSeedNode("MiningStat", nil)
	}()

	go n.NetWork.LoadStatus.HandleBlock(n.NodeState.LatestBlock)
}

func (n *Node) HandlePrePrepare(msg *Message.NodeMsg) {
	n.NodeState.Stage = "prepare"
	n.NodeState.BlockHeight = msg.BlockHeight
	if ID.StringToID(msg.From).ShardID == n.NodeState.NodeID.ShardID {
		n.SendPrepare(n.NodeState.NodeID.ShardID)
	} else {
		n.SendPrepare(1024)
	}
}

func (n *Node) SendPrepare(shardId int) {
	msg := Message.NewNodeMsg(n.NetWork.ServerID, "", "Prepare", nil)
	msg.BlockHeight = n.NodeState.BlockHeight
	n.BroadCastWithInShard(shardId, msg)

}

func (n *Node) HandlePrepare(msg *Message.NodeMsg) {

	var shardMemNum int
	var shardType string
	//不同分片成员数量不同
	if ID.StringToID(msg.From).ShardID == n.NodeState.NodeID.ShardID {
		shardMemNum = len(n.NetWork.Peers.Shards[n.NodeState.NodeID.ShardID].Member)
		shardType = "IntraPrepare"
	} else {
		shardMemNum = len(n.NetWork.Peers.Shards[1024].Member)
		shardType = "GlobalPrepare"
	}

	finish, quorum := n.NodeState.PrepareCheck[shardType].Handle(Malicious, shardMemNum, msg)

	if finish {
		n.Logger.Debugf(
			"NodeID: %s, get enough %s msg, quorum is %d, TotalNodeNum: %d",
			n.NetWork.ServerID,
			shardType,
			quorum,
			shardMemNum,
		)
		n.NodeState.PrepareCheck[shardType].Finish = true

		if shardType == "IntraPrepare" {
			n.SendCommit(n.NodeState.NodeID.ShardID)
		} else {
			n.SendCommit(1024)
		}

	} else {
		if quorum == 1024 {

		} else {
			n.Logger.Debugf(
				"NodeID: %s, waiting for %s msg, quorum is %d, TotalNodeNum: %d, now %d votes",
				n.NetWork.ServerID,
				shardType,
				quorum,
				shardMemNum,
				n.NodeState.PrepareCheck[shardType].VoteCount,
			)
		}
	}

}

func (n *Node) SendCommit(shardId int) {
	msg := Message.NewNodeMsg(n.NetWork.ServerID, "", "Commit", nil)
	msg.BlockHeight = n.NodeState.BlockHeight
	n.BroadCastWithInShard(shardId, msg)
}

func (n *Node) HandleCommit(msg *Message.NodeMsg) {

	var shardMemNum int
	var shardType string
	//不同分片成员数量不同
	if ID.StringToID(msg.From).ShardID == n.NodeState.NodeID.ShardID {
		shardMemNum = len(n.NetWork.Peers.Shards[n.NodeState.NodeID.ShardID].Member)
		shardType = "IntraCommit"
	} else {
		shardMemNum = len(n.NetWork.Peers.Shards[1024].Member)
		shardType = "GlobalCommit"
	}

	finish, quorum := n.NodeState.CommitCheck[shardType].Handle(Malicious, shardMemNum, msg)

	if finish {
		n.Logger.Debugf(
			"NodeID: %s, get enough %s msg, quorum is %d, TotalNodeNum: %d",
			n.NetWork.ServerID,
			shardType,
			quorum,
			shardMemNum,
		)
		n.NodeState.CommitCheck[shardType].Finish = true
		if n.NodeState.Role == "GlobalLeader" {
			//自己不发送自己
		} else {
			n.DialGlobalLeader("Finality", nil)
		}

	} else {
		if quorum == 1024 {

		} else {
			n.Logger.Debugf(
				"NodeID: %s, waiting for %s msg, quorum is %d, TotalNodeNum: %d, now %d votes",
				n.NetWork.ServerID,
				shardType,
				quorum,
				shardMemNum,
				n.NodeState.CommitCheck[shardType].VoteCount,
			)
		}
	}

}

func (n *Node) HandleFinality(msg *Message.NodeMsg) {

	var shardMemNum int
	var shardType string
	//不同分片成员数量不同
	if ID.StringToID(msg.From).ShardID == n.NodeState.NodeID.ShardID {
		shardMemNum = len(n.NetWork.Peers.Shards[n.NodeState.NodeID.ShardID].Member)
		shardType = "IntraFinality"
	} else {
		shardMemNum = len(n.NetWork.Peers.Shards[1024].Member)
		shardType = "GlobalFinality"
	}

	finish, quorum := n.NodeState.FinalityCheck[shardType].Handle(Malicious, shardMemNum, msg)

	if finish {
		n.Logger.Debugf(
			"NodeID: %s, get enough %s msg, quorum is %d, TotalNodeNum: %d",
			n.NetWork.ServerID,
			shardType,
			quorum,
			shardMemNum,
		)
		n.NodeState.FinalityCheck[shardType].Finish = true
		//判断是否为片外达成共识
		if shardType == "GlobalFinality" {
			n.NextRound()
		} else {
			//片外未达成共识，继续等待
		}

	} else {
		if quorum == 1024 {

		} else {
			n.Logger.Debugf(
				"NodeID: %s, waiting for %s msg, quorum is %d, TotalNodeNum: %d, now %d votes",
				n.NetWork.ServerID,
				shardType,
				quorum,
				shardMemNum,
				n.NodeState.FinalityCheck[shardType].VoteCount,
			)
		}
	}

}

func (n *Node) NextRound() {
	n.Logger.Debugf(
		"NodeID: %s, Round %d is finished",
		n.NetWork.ServerID,
		n.NodeState.BlockHeight,
	)

	if n.NodeState.BlockHeight == EndBlockHeight {
		close(n.NodeCh.TxWorkerStop)
		//n.NetWork.LoadStatus.EndTime = time.Now()
		n.DialSeedNode("MiningStop", nil)
	} else {
		n.StartConsensus()
	}
}
