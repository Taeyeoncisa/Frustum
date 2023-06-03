package SS_PBFT_V1

import (
	"BlockChain/CLI"
	"BlockChain/Consensus/PBFT"
	"BlockChain/ID"
	"BlockChain/Message"
	"BlockChain/NetWork"
	"BlockChain/TxPool"
	"go.uber.org/zap"
)

type Node struct {
	Logger *zap.SugaredLogger

	NodeState *State
	NodeCh    *ChSum
	PBFTNode  *PBFT.ConsensusNode

	NodeTxPool *TxPool.TxPool
	NodeNet    *NetWork.Server
}

func NewNode(logger *zap.SugaredLogger, nodeID ID.NodeID, ipAddr string) *Node {

	node := Node{
		Logger:     logger,
		NodeState:  NewState(),
		NodeCh:     NewChSum(),
		PBFTNode:   PBFT.NewPBFTNode(logger, nodeID.String()),
		NodeTxPool: TxPool.NewTxPool(false, 3),
		NodeNet:    nil,
	}

	node.NodeNet = NetWork.NewServer(logger, nodeID, node.NodeCh.ReceiveMsgCh, node.NodeCh.SendMsgCh)

	return &node
}

func (n *Node) Run() {
	// 开启网络服务
	go n.NodeNet.Start()
	// 开启命令行服务
	go CLI.Input(n.NodeCh.CLI)
	// 开启消息发送管道监听
	go n.NodeNet.WaitMsgToSend()
	// 共识节点启动后向种子节点注册
	go n.Registration()
	//节点主循环
	for {
		select {
		case <-n.NodeCh.ElegantExitCh:
			//告知server退出
			n.NodeNet.ElegantExitCh <- struct{}{}
			//节点主循环退出
			n.Logger.Debug("node gracefully shutdown")
			return
		case msg := <-n.NodeCh.ReceiveMsgCh:
			n.MsgHandle(msg)
		case cli := <-n.NodeCh.CLI:
			n.CLIHandle(cli)
		}

	}
}

func (n *Node) CLIHandle(cli string) {
	//n.Logger.Debug(cli)
	switch cli {
	case "quit":
		n.NodeCh.ElegantExitCh <- struct{}{}
	}
}

func (n *Node) Registration() {
	seedNode := ID.NodeID{
		ShardID:   0,
		InShardID: 0,
	}
	msg := Message.NewNodeMsg(n.NodeNet.SelfNodeID, seedNode, "registration", n.NodeNet.Ipaddr)

	n.NodeCh.SendMsgCh <- msg

}

func (n *Node) LengthOfShard(shardId int) int {
	return len(n.NodeNet.Shards.Shards[shardId].Member)
}
