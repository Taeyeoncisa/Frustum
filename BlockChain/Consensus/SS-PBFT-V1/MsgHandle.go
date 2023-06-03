package SS_PBFT_V1

import (
	"BlockChain/ID"
	"BlockChain/Message"
	"BlockChain/NetWork"
	"fmt"
	"time"
)

func (n *Node) MsgHandle(msg *Message.NodeMsg) {
	switch msg.MsgType {

	case "start":
		go n.NodeTxPool.CycleAddTx()
		//n.StartMining()
		n.StartConsensus()

	case "PrePrepare":
		newBlock := msg.UnmarshalBlock()
		n.Logger.Debugf("NodeID: %s, MsgType: %s, sender's EpochNum: %d, BlockHeight: %d, My BlockHeight: %d",
			n.NodeNet.SelfNodeID.String(), msg.MsgType, msg.EpochNum, newBlock.BlockHeight, n.NodeState.BlockHeight)

		if msg.BlockHeight > n.NodeState.BlockHeight {
			n.NodeState.BlockHeight = msg.BlockHeight
			n.NodeState.LatestBlock = newBlock

			/**************************************************/
			n.ResetConsensusCheckCount()
			/**************************************************/

			respondMsg := Message.NewNodeMsg(n.NodeNet.SelfNodeID, msg.From,
				"Prepare", nil)
			respondMsg.EpochNum = n.NodeState.NodeEpoch
			respondMsg.BlockHeight = n.NodeState.BlockHeight
			n.BroadCast("shard", n.NodeNet.SelfNodeID.ShardID, respondMsg)
		} else {
			//maybe do something
		}

	case "Prepare":
		n.Logger.Debugf("NodeID: %s, MsgType: %s,  sender's EpochNum: %d, BlockHeight: %d, My BlockHeight: %d",
			n.NodeNet.SelfNodeID.String(), msg.MsgType, msg.EpochNum, msg.BlockHeight, n.NodeState.BlockHeight)
		ok, quorum := n.NodeState.PrepareCheck.Handle(QuorumPercentage, n.LengthOfShard(n.NodeNet.SelfNodeID.ShardID), msg.BlockHeight)
		if ok {
			n.Logger.Debugf("NodeID: %s, get enough respond in PrepareCheck, quorum: %d, TotalMem: %d",
				n.NodeNet.SelfNodeID.String(), quorum, n.LengthOfShard(n.NodeNet.SelfNodeID.ShardID))
			n.SendCommit()
		} else {
			//由于pre-prepare消息过大，导致prepare消息先抵达follower，出现prepare消息轮次过大被丢弃的问题
			//简单处理办法：直接将消息扔回接收管道直至被积极处理
			if quorum == 1024 {
				go n.WaitSomeTime(msg)
			}
		}

	case "Commit":
		ok, quorum := n.NodeState.CommitCheck.Handle(QuorumPercentage, n.LengthOfShard(n.NodeNet.SelfNodeID.ShardID), msg.BlockHeight)
		if ok {
			n.Logger.Debugf("NodeID: %s, get enough respond in CommitCheck, quorum: %d, TotalMem: %d",
				n.NodeNet.SelfNodeID.String(), quorum, n.LengthOfShard(n.NodeNet.SelfNodeID.ShardID))

			if n.NodeState.NodeRole == "follower" {
				n.SendFinalToIntraLeader()
			} else {
				//仅follower反馈确认至leader
			}

		} else {

		}

	case "FinalCommit":
		ok, quorum := n.NodeState.FinalCommitCheck.Handle(QuorumPercentage, n.LengthOfShard(n.NodeNet.SelfNodeID.ShardID), msg.BlockHeight)
		if ok {
			n.Logger.Debugf("NodeID: %s, get enough respond in FinalCommitCheck, quorum: %d, TotalMem: %d",
				n.NodeNet.SelfNodeID.String(), quorum, n.LengthOfShard(n.NodeNet.SelfNodeID.ShardID))

			if n.NodeState.BlockHeight < StopMiningBlockHeight {
				n.StartConsensus()
			} else {
				n.Logger.Infof("NodeID: %s, stop mining, BlockHeight: %d",
					n.NodeNet.SelfNodeID.String(), n.NodeState.BlockHeight)
				n.NodeNet.LoadStats.PrintResult()
				n.InformSeedNodeMiningStop()
			}

		} else {

		}

	case "NodeMember":
		nodeMem := msg.UnmarshalNodeMem()

		n.NodeNet.Shards.AddShard(n.NodeMemProcessing(msg.MsgType, nodeMem))

		for k, v := range n.NodeNet.Shards.Shards {
			fmt.Println("ShardID:", k, v.Member)
		}
	case "IntraLeaderMem":
		nodeMem := msg.UnmarshalNodeMem()

		n.NodeNet.Shards.AddShard(n.NodeMemProcessing(msg.MsgType, nodeMem))

		for k, v := range n.NodeNet.Shards.Shards {
			fmt.Println("ShardID:", k, v.Member)
		}
		for k := range *nodeMem {
			if n.NodeNet.SelfNodeID.String() == k.String() {
				n.NodeState.NodeRole = "intraLeader"
				n.Logger.Infof("BecomeIntraLeader, NodeID: %s", n.NodeNet.SelfNodeID.String())
			} else {
				continue
			}
		}
	case "GlobalLeaderMem":
		nodeMem := msg.UnmarshalNodeMem()

		n.NodeNet.Shards.AddShard(n.NodeMemProcessing(msg.MsgType, nodeMem))

		for k, v := range n.NodeNet.Shards.Shards {
			fmt.Println("ShardID:", k, v.Member)
		}
		for k := range *nodeMem {
			if n.NodeNet.SelfNodeID.String() == k.String() {
				n.NodeState.NodeRole = "GlobalLeader"
				n.Logger.Infof("BecomeGlobalLeader, NodeID: %s", n.NodeNet.SelfNodeID.String())
			} else {
				continue
			}
		}

	case "block":
		newBlock := msg.UnmarshalBlock()
		n.NodeState.LatestBlock = newBlock
		if n.NodeState.NodeRole == "intraLeader" {
			msgWithBlock := Message.NewNodeMsg(n.NodeNet.SelfNodeID, n.NodeNet.SelfNodeID,
				"block", n.NodeState.LatestBlock)
			n.BroadCastV2("shard", n.NodeNet.SelfNodeID.ShardID, msgWithBlock)
		}

	case "ShutDown":
		n.NodeCh.ElegantExitCh <- struct{}{}
	}
}

// NodeMemProcessing 该函数解析seedNode发送过来的分片成员信息
func (n *Node) NodeMemProcessing(shardType string, shard *map[ID.NodeID]string) (int, *NetWork.MemberManager) {

	shardID := 0
	memManger := new(NetWork.MemberManager)
	memManger.Member = *shard

	switch shardType {
	case "IntraLeaderMem":
		shardID = 1024
	case "GlobalLeaderMem":
		shardID = 1025
	default:
		for k := range *shard {
			shardID = k.ShardID
		}
	}

	return shardID, memManger
}

// BroadCast 节点广播消息函数，可选广播范围，传入的msg无须填充目标地址
func (n *Node) BroadCast(scope string, shard int, msg *Message.NodeMsg) {

	switch scope {

	case "all":

		for ShardID, SHARD := range n.NodeNet.Shards.Shards {

			switch ShardID {

			case 0, 1024, 1025:
				//0，1024为特殊分片ID
			default:
				for k := range SHARD.Member {

					if k.String() == n.NodeNet.SelfNodeID.String() {
						//自己就不发送
					} else {
						newMsg := *msg
						newMsg.To = k

						n.NodeCh.SendMsgCh <- &newMsg
					}

				}
			}

		}

	case "shard":

		nodes := n.NodeNet.Shards.Shards[shard]

		for k := range nodes.Member {

			if k.String() == n.NodeNet.SelfNodeID.String() {
				//自己就不发送
			} else {
				newMsg := *msg
				newMsg.To = k

				n.NodeCh.SendMsgCh <- &newMsg
			}

		}

	}

}

func (n *Node) BroadCastV2(scope string, shard int, msg *Message.NodeMsg) {

	switch scope {

	case "all":

		for ShardID, SHARD := range n.NodeNet.Shards.Shards {

			switch ShardID {

			case 0, 1024, 1025:
				//0，1024为特殊分片ID
			default:
				for k := range SHARD.Member {

					if k.String() == n.NodeNet.SelfNodeID.String() {
						//自己就不发送
					} else {
						newMsg := *msg
						newMsg.To = k

						n.NodeNet.SendNodeMsg(n.NodeNet.Shards.FindNode(k), &newMsg)
					}

				}
			}

		}

	case "shard":

		nodes := n.NodeNet.Shards.Shards[shard]

		for k := range nodes.Member {

			if k.String() == n.NodeNet.SelfNodeID.String() {
				//自己就不发送
			} else {
				newMsg := *msg
				newMsg.To = k

				n.NodeNet.SendNodeMsg(n.NodeNet.Shards.FindNode(k), &newMsg)
			}

		}

	}

}

func (n *Node) WaitSomeTime(msg *Message.NodeMsg) {
	select {
	case <-time.After(60 * time.Millisecond):
		n.NodeCh.ReceiveMsgCh <- msg
	}
}
