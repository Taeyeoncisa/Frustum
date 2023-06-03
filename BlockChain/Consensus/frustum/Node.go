package SS_PBFT_V2

import (
	"BlockChain/CLI"
	"BlockChain/Message"
	"BlockChain/NetWork"
	"BlockChain/TxPool"
	"fmt"
	"go.uber.org/zap"
)

const Malicious = float64(1) / 3

type Node struct {
	Logger *zap.SugaredLogger

	NodeState  *State
	NodeTxPool *TxPool.TxPool
	NodeCh     *ChSum

	NetWork *NetWork.Server
}

func NewNode(logger *zap.SugaredLogger, nodeID, ipAddr string) *Node {

	nodeCh := NewChSum()
	txPool := TxPool.NewTxPool(false, 3)

	peers := NetWork.NewNodeTable(nodeID)
	server := NetWork.NewServer(logger, nodeID, peers, nodeCh.RecMsgCh, nodeCh.SenMsgCh, nodeCh.MainProcessExit, txPool)

	node := Node{
		Logger:     logger,
		NodeState:  NewState(nodeID),
		NodeTxPool: txPool,
		NodeCh:     nodeCh,
		NetWork:    server,
	}

	return &node
}

func (n *Node) Run() {

	go n.NetWork.Start()
	go CLI.Input(n.NodeCh.CLI)
	go n.DialSeedNode("registration", n.NetWork.HttpServer.Addr)

	for {
		select {
		case <-n.NodeCh.MainProcessExit:
			n.Logger.Debug("Node gracefully shutdown")
			return
		case msg := <-n.NodeCh.RecMsgCh:
			n.MSGHandle(msg)
		case cli := <-n.NodeCh.CLI:
			n.CLIHandle(cli)
		}
	}

}

func (n *Node) CLIHandle(cli string) {

	switch cli {
	case "quit":
		close(n.NetWork.ElegantExitCh)
	}
}

func (n *Node) MSGHandle(msg *Message.NodeMsg) {

	switch msg.MsgType {

	//This is the consensus
	/*********************************************************************************/
	/*********************************************************************************/
	/*********************************************************************************/

	case "start":
		go n.NodeTxPool.CycleAddTx(n.NodeCh.TxWorkerStop)
		n.StartConsensus()
	case "PrePrepare":
		n.ResetVoteCheck(msg.BlockHeight)
		n.HandlePrePrepare(msg)
	case "Prepare":
		n.HandlePrepare(msg)
	case "Commit":
		n.HandleCommit(msg)
	case "Finality":
		n.HandleFinality(msg)

	/*********************************************************************************/
	/*********************************************************************************/
	/*********************************************************************************/

	//Network initialization and node discovery
	case "NodeMember", "IntraLeaderMem", "GlobalLeaderMem":
		shard := msg.UnmarshalShardMem()
		n.WhetherOrNotLeader(n.NetWork.Peers.HandleShardMsg(msg.MsgType, shard))

	case "ShutDown":
		close(n.NetWork.ElegantExitCh)
	}

}

func (n *Node) DialSeedNode(msgType string, msgBody interface{}) {

	msg := Message.NewNodeMsg(n.NetWork.ServerID, "0-0", msgType, msgBody)
	msg.BlockHeight = n.NodeState.BlockHeight
	n.NodeCh.SenMsgCh <- msg

}

func (n *Node) DialGlobalLeader(msgType string, msgBody interface{}) {

	ok, globalLeader, _ := n.NetWork.Peers.FindMyGlobalLeader()
	if ok {
		msg := Message.NewNodeMsg(
			n.NetWork.ServerID,
			globalLeader,
			msgType,
			msgBody,
		)
		msg.BlockHeight = n.NodeState.BlockHeight
		n.NodeCh.SenMsgCh <- msg
	} else {
		n.Logger.Debugf(
			"NodeID: %s, GlobalLeader not Exist",
			n.NetWork.ServerID,
		)
	}

}

func (n *Node) WhetherOrNotLeader(shardID int, leader bool) {
	if leader {

		switch shardID {

		case 1024:
			n.NodeState.Role = "IntraLeader"
			n.Logger.Debugf("NodeID: %s, become %s", n.NetWork.ServerID, n.NodeState.Role)
		case 1025:
			n.NodeState.Role = "GlobalLeader"
			n.Logger.Debugf("NodeID: %s, become %s", n.NetWork.ServerID, n.NodeState.Role)
		}

	} else {

	}

	for k, v := range n.NetWork.Peers.Shards {
		fmt.Printf("ShardID %d Members:\n", k)
		for nodeId, url := range v.Member {
			fmt.Printf("[%s %s]\n", nodeId, url)
		}
	}
}

func (n *Node) BroadCastWithInShard(shardId int, msg *Message.NodeMsg) {

	shard := n.NetWork.Peers.Shards[shardId]
	for node, url := range shard.Member {

		if node == n.NetWork.ServerID {
			//自己就不发送
		} else {
			newMsg := *msg
			newMsg.To = node
			go n.NetWork.SendNodeMsg(url, &newMsg)
		}

	}

}
