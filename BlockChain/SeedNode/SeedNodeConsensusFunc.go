package SeedNode

import (
	"BlockChain/Message"
)

// GiveStartToGlobalLeader 种子节点启动共识
func (n *SeedNode) GiveStartToGlobalLeader() {

	var globalLeader string

	for k := range n.Peers.Shards[1025].Member {
		globalLeader = k
	}

	msg := Message.NewNodeMsg(n.Peers.SelfNodeID, globalLeader, "start", nil)
	n.SeedNodeCh.SendMsgCh <- msg
}

func (n *SeedNode) GiveStartToIntraLeaders() {

	for k := range n.Peers.Shards[1024].Member {

		msg := Message.NewNodeMsg(n.Peers.SelfNodeID, k, "start", nil)
		n.SeedNodeCh.SendMsgCh <- msg

	}

}
