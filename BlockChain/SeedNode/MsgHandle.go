package SeedNode

import (
	"BlockChain/Message"
	"time"
)

func (n *SeedNode) MSGHandle(msg *Message.NodeMsg) {

	switch msg.MsgType {

	case "registration":
		//当前共识算法类型
		n.SeedNodeState.ConsensusType = msg.ConsensusTye
		//注册远程节点的ID和IP
		n.Peers.AddNode(msg.From, msg.UnmarshalClientAddr())

	case "MiningStat":
		n.Logger.Debugf("Miner:%s, BlockHeight: %d", msg.From, msg.BlockHeight)
	case "IntraMiningStat":
		n.Logger.Debugf("Miner:%s, IntraBlockHeight: %d", msg.From, msg.BlockHeight)
	case "MiningStop":
		n.Logger.Debugf("Miner:%s, Mining Finish", msg.From)

		switch n.SeedNodeState.ConsensusType {

		case "RapidChain":
			n.SeedNodeState.ConsensusStop++
			if n.SeedNodeState.ConsensusStop == len(n.Peers.Shards[1024].Member) {
				time.Sleep(1 * time.Second)
				n.SeedNodeCh.CLI <- "exit"
			} else {
				//not all the shards finish
			}

		default:
			time.Sleep(1 * time.Second)
			n.SeedNodeCh.CLI <- "exit"
		}

	}

}
