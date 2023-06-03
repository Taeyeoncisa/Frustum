package Node

import (
	"BlockChain/DB"
	"BlockChain/Message"
	"BlockChain/NetWork"
	"BlockChain/TxPool"
	"bytes"
	"fmt"
	"go.uber.org/zap"
)

type Node struct {
	Logger *zap.Logger

	NodeState *State
	NodeCh    *ChSum

	NodeTxPool *TxPool.TxPool
	NodeBC     *DB.BlockChain

	NodeNet *NetWork.Server

	HeatBeatTimeOut  *Tick
	CandidateTimeOut *Tick
}

func NewNode(logger *zap.Logger,
	nodeId, consensus string,
	doneCh chan struct{}) *Node {
	var priority bool
	if consensus == "Raft" {
		priority = false
	} else {
		priority = true
	}

	node := Node{
		Logger:           logger,
		NodeState:        NewState(consensus),
		NodeCh:           NewChSum(doneCh),
		NodeTxPool:       TxPool.NewTxPool(priority, 3),
		NodeBC:           DB.NewBlockChain(nodeId),
		NodeNet:          nil,
		HeatBeatTimeOut:  NewNodeTick(),
		CandidateTimeOut: NewNodeTick(),
	}

	return &node
}

func (n *Node) Run() {

	for {
		select {

		case <-n.NodeCh.ElegantExitCh:
			n.Logger.Info(n.NodeNet.SelfNodeID.String() + " Exit!")
			return
		case <-n.NodeCh.Q1CheckTrigger:
			go n.CheckConfirmStartVote()

		case <-n.NodeCh.Q2CheckTrigger:
			switch n.NodeState.ConsensusType {
			case "SS-Raft":
				go n.SSRaftCheckCommit()
			case "Raft":
				go n.RaftCheckCommit()
			}

		case msg := <-n.NodeNet.RecMsgCh:
			//n.Logger.Debug(n.NodeID.String() + " enter HandleMsg " + msg.MsgType)
			n.HandleMsg(msg)
			//n.Logger.Debug(n.NodeID.String() + " Finish HandleMsg " + msg.MsgType)

		}
	}

}

func (n *Node) HandleMsg(msg *Message.NodeMsg) {

	//info := fmt.Sprintf("%s get Msg From %s %s",
	//	n.NodeID.String(),
	//	msg.NodeID.String(),
	//	msg.MsgType)
	//n.Logger.Debug(info)

	switch msg.MsgType {
	case "HelloWorld":
		n.Logger.Info(n.NodeNet.SelfNodeID.String() + " HelloWorld")
	case "GlobalMem":
		leadersID := msg.UnmarshalGlobalMem()
		n.GetGlobalMem(leadersID)
	/*******************************************/
	case "start":
		//n.Logger.Debug("Get startSignal")
		go n.HeartBeatTimeOutDetection()
	case "StartVote":
		n.RespondStartVote(msg)
	case "ConfirmStartVote":
		n.NodeCh.Q1CheckMsgCh <- msg

	case "HeartBeat":
		n.NodeState.IntraLeader = msg.From

		//info := fmt.Sprintf("%s get HeartBeat, RcMsgChLength:%d",
		//	n.NodeID.String(), len(n.NodeCh.ReceiveMsgCh))
		//n.Logger.Debug(info)

		switch n.NodeState.IntraRole {
		case "intraFollower":
			n.HeatBeatTimeOut.StartOrRestTick(intraHeartBeatTimeOut)
		case "intraCandidate":
			n.NodeState.IntraRole = "intraFollower"
			n.Logger.Debug(n.NodeNet.SelfNodeID.String() + " back to intraFollower")
			n.HeatBeatTimeOut.Run(intraHeartBeatTimeOut)
		}
	case "Transaction":

		if n.NodeState.IntraRole == "intraLeader" {
			tr := msg.UnmarshalTr()
			n.NodeTxPool.AddTr(tr)
		} else {
			n.NodeNet.SendNodeMsg(n.NodeNet.AllNode.Member[n.NodeState.IntraLeader], msg)
		}
	case "StartCommit":
		n.Logger.Info(n.NodeNet.SelfNodeID.String() + " receive StartCommit, consensus: " + n.NodeState.ConsensusType)

		switch n.NodeState.ConsensusType {
		case "Raft":
			n.RaftStartMining()
		case "SS-Raft":
			if msg.MsgBody != nil {
				if ok := bytes.Equal(msg.MsgBody, n.NodeBC.Tip); ok {
					n.Logger.Debug("BlockHash in StartCommit is same with me")
				} else {
					n.NodeBC.Tip = msg.MsgBody
				}
			} else {
				n.Logger.Debug("No BlockHash in StartCommit")
			}
			n.SSRaftStartMining()
		}
	case "block":
		block := msg.UnmarshalBlock()
		info := fmt.Sprintf("%s Consensus: %s, receive No.%d Block from %s",
			n.NodeNet.SelfNodeID.String(),
			n.NodeState.ConsensusType,
			block.BlockHeight,
			msg.From.String())
		n.Logger.Info(info)

		n.NodeState.BlockHeight++
		if n.NodeState.BlockHeight < block.BlockHeight {
			n.NodeState.BlockHeight = block.BlockHeight
		}

		n.NodeBC.AddNewBlock(block)

		msgR := Message.NodeMsg{
			TimeStamp: 0,
			From:      n.NodeNet.SelfNodeID,
			To:        msg.From,
			MsgType:   "ConfirmBlock",
			MsgBody:   nil,
			//IntraTerm:   n.NodeState.IntraTerm,
			//BlockHeight: n.NodeState.BlockHeight,
		}

		n.NodeNet.SendNodeMsg(n.NodeNet.AllNode.Member[msgR.To], &msgR)
	/*case "blockHash":
	info := fmt.Sprintf("%s Consensus: %s, receive No.%d PreComBlock from %s",
		n.NodeNet.SelfNodeID.String(),
		n.NodeState.ConsensusType,
		msg.BlockHeight,
		msg.From.String())
	n.Logger.Info(info)

	n.NodeState.BlockHeight++
	if n.NodeState.BlockHeight < msg.BlockHeight {
		n.NodeState.BlockHeight = msg.BlockHeight
	}

	n.NodeBC.AddNewBlockHash(msg.MsgBody)

	msgR := Message.NodeMsg{
		TimeStamp:   0,
		From:        n.NodeNet.SelfNodeID,
		To:          msg.From,
		MsgType:     "ConfirmBlock",
		MsgBody:     nil,
		IntraTerm:   n.NodeState.IntraTerm,
		BlockHeight: n.NodeState.BlockHeight,
	}

	n.NodeNet.SendMessage(msgR.To, &msgR)*/
	case "ConfirmBlock":
		//info2 := fmt.Sprintf("%s ConfirmBlock from %s, BlockHeight:%d",
		//	n.NodeID.String(),
		//	statMsg.NodeId.String(),
		//	statMsg.BlockHeight)
		//n.Logger.Debug(info2)

		n.NodeCh.Q2CheckMsgCh <- msg

	/********************************************/
	case "analysis":
		result := n.NodeBC.CalculateTPSandLatency()
		if result {
			close(n.NodeCh.ElegantExitCh)
		}
	//case "readTest":
	//	n.BroadCastMessage("global", "ping", nil)
	//	n.BroadCastMessage("intra", "ping", nil)
	//case "ping":
	//	n.Logger.Debug(n.NodeID.String() + " receive ping from " + msg.NodeID.String())
	//	_ = n.NodeBC.ReadTip()
	//	n.SendMessage(msg.NodeID, "pong", nil)
	//case "pong":
	//	n.Logger.Debug(n.NodeID.String() + " receive pong from " + msg.NodeID.String())

	/********************************************/
	default:
		n.Logger.Info(n.NodeNet.SelfNodeID.String() + " Don't know MsgType")

	}
}
