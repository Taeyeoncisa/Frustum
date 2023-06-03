package Node

import (
	"SS-Raft/DB"
	"SS-Raft/ID"
	"SS-Raft/Message"
	"SS-Raft/NetWork"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"
)

const intraHeartBeatCycleTime = 200
const intraHeartBeatTimeOut = 2000

const TestingMaxBlockHeight = 128
const BatchBlocksNum = 128

func check(err error) {

	if err != nil {
		log.Panic(err)
	}

}

func (n *Node) GetGlobalMem(leaders *[]ID.NodeID) {
	n.Logger.Info(n.NodeNet.SelfNodeID.String() + " Get GlobalMemIDs")
	for _, v := range *leaders {

		n.NodeNet.ShardLeaders.Member[v] = n.NodeNet.AllNode.Member[v]

	}

}

// HeartBeatTimeOutDetection 心跳超时计时器
func (n *Node) HeartBeatTimeOutDetection() {

	n.HeatBeatTimeOut.Run(intraHeartBeatTimeOut)
	select {
	case <-n.HeatBeatTimeOut.TimeOut:
		info := fmt.Sprintf("%s HeartBeatTimeOut: %dms",
			n.NodeNet.SelfNodeID.String(),
			n.HeatBeatTimeOut.CycleTime)
		n.Logger.Info(info)
		n.NodeState.IntraRole = "intraCandidate"
		n.NodeState.IntraTerm++
		//分片后片内和片外延迟不同，Raft仍全网广播
		//SS-Raft仅在片内实施
		switch n.NodeState.ConsensusType {
		case "Raft":
			scale := *n.NodeNet.AllNode
			delete(scale.Member, ID.NodeID{
				ShardID:   0,
				InShardID: 0,
			})
			n.SendStartVote(&scale)
		default:
			n.SendStartVote(n.NodeNet.Shards[n.NodeNet.SelfNodeID.ShardID])
		}
	}

}

// CycleHeartBeat 循环心跳发送
func (n *Node) CycleHeartBeat(cycleTime int, scale *NetWork.MemberManager) {

	ticker := time.NewTicker(time.Duration(cycleTime) * time.Millisecond)
	defer ticker.Stop()

	nodeMsg := Message.NodeMsg{
		TimeStamp:   0,
		From:        ID.NodeID{},
		To:          ID.NodeID{},
		MsgType:     "HeartBeat",
		MsgBody:     nil,
		IntraTerm:   0,
		BlockHeight: 0,
	}
	n.NodeNet.BroadCastMessage(scale, &nodeMsg)
	for {

		select {
		case <-n.NodeCh.IntraStopHeartCh:
			return
		case <-ticker.C:
			n.NodeNet.BroadCastMessage(scale, &nodeMsg)
		}

	}

}

/************************************************************/

func (n *Node) SendStartVote(scale *NetWork.MemberManager) {

	msg := Message.NodeMsg{
		TimeStamp:   0,
		From:        ID.NodeID{},
		To:          ID.NodeID{},
		MsgType:     "StartVote",
		MsgBody:     nil,
		IntraTerm:   n.NodeState.IntraTerm,
		BlockHeight: n.NodeState.BlockHeight,
	}

	n.NodeNet.BroadCastMessage(scale, &msg)
	n.NodeCh.Q1CheckTrigger <- struct{}{}
}

func (n *Node) RespondStartVote(msg *Message.NodeMsg) {
	info1 := fmt.Sprintf("%s StartVote From: %s, startVoteTerm: %v; selfNowTerm: %v; latestVoteTerm: %v",
		n.NodeNet.SelfNodeID.String(),
		msg.From.String(),
		msg.IntraTerm,
		n.NodeState.IntraTerm,
		n.NodeState.IntraCandiStat.Term)
	n.Logger.Debug(info1)
	//重置计时器
	switch n.NodeState.IntraRole {
	case "intraFollower":
		n.HeatBeatTimeOut.StartOrRestTick(intraHeartBeatTimeOut)
	case "intraCandidate":
		n.Logger.Debug(n.NodeNet.SelfNodeID.String() + " I'm intraCandidate, don't reset TimeOutDetection")

	}
	//发起者term高于自己的term，且自己未在发起者term投过票，且区块信息比较新
	if msg.IntraTerm > n.NodeState.IntraTerm &&
		msg.BlockHeight >= n.NodeState.BlockHeight &&
		msg.IntraTerm > n.NodeState.IntraCandiStat.Term {
		info2 := fmt.Sprintf("%s I'm %s, get a Higher TermNum %d from %s",
			n.NodeNet.SelfNodeID.String(),
			n.NodeState.IntraRole,
			msg.IntraTerm,
			msg.From.String())
		n.Logger.Debug(info2)

		n.NodeState.IntraTerm = msg.IntraTerm
		n.NodeState.IntraCandiStat.Term = msg.IntraTerm
		n.NodeState.IntraCandiStat.Id = msg.From

		msgConfirm := Message.NodeMsg{
			TimeStamp:   0,
			From:        ID.NodeID{},
			To:          ID.NodeID{},
			MsgType:     "ConfirmStartVote",
			MsgBody:     nil,
			IntraTerm:   n.NodeState.IntraTerm,
			BlockHeight: n.NodeState.BlockHeight,
		}

		n.NodeNet.SendMessage(msg.From, &msgConfirm)
	} else {
		n.Logger.Debug(n.NodeNet.SelfNodeID.String() + " don't respond StartVote!")
		// nothing
	}
}

func (n *Node) CheckConfirmStartVote() {
	var Q1 uint
	//模拟网络分片后仍使用Raft算法，Q1根据全网节点数量计算
	//SS-Raft则只按片内大小计算
	switch n.NodeState.ConsensusType {
	case "Raft":
		Q1 = uint(math.Floor(float64(len(n.NodeNet.AllNode.Member)-1) * 0.5))
	default:
		Q1 = uint(math.Floor(float64(len(n.NodeNet.Shards[n.NodeNet.SelfNodeID.ShardID].Member)) * 0.5))
	}

	n.Logger.Info("Q1 is: " + strconv.Itoa(int(Q1)))
	for {

		select {
		case msg := <-n.NodeCh.Q1CheckMsgCh:
			if msg.IntraTerm < n.NodeState.IntraTerm {
				//历史消息，不理会
				continue
			} else {
				info1 := fmt.Sprintf("%s ConfirmStartVote From: %s, MsgTerm: %v; selfNowTerm: %v",
					n.NodeNet.SelfNodeID.String(),
					msg.From.String(),
					msg.IntraTerm,
					n.NodeState.IntraTerm)
				n.Logger.Debug(info1)
			}
			//重置清空历史计数消息
			if n.NodeState.IntraQ1Count.Term < n.NodeState.IntraTerm {
				n.NodeState.IntraQ1Count.Count = 0
				n.NodeState.IntraQ1Count.Term = n.NodeState.IntraTerm
			} else {
				// do something
			}

			if n.NodeState.IntraQ1Count.Count < Q1 {
				n.NodeState.IntraQ1Count.Count++
				if n.NodeState.IntraQ1Count.Count == Q1 {
					switch n.NodeState.IntraRole {

					case "intraCandidate":
						n.BecomeIntraLeader()
					default:
						n.Logger.Debug("no right to become leader")

					}
					return
				} else {
					continue
				}
			} else {
				n.Logger.Debug(n.NodeNet.SelfNodeID.String() + " error in CheckConfirmStartVote ")
			}

		case <-n.NodeCh.Q1CheckBreak:
			return
		}
	}

}

func (n *Node) BecomeIntraLeader() {

	n.NodeState.IntraRole = "intraLeader"
	n.NodeState.GlobalRole = "globalFollower"
	n.Logger.Info(n.NodeNet.SelfNodeID.String() + " become intraLeader")

	//分片后片内和片外延迟不同，Raft仍全网广播
	//SS-Raft仅在片内实施
	switch n.NodeState.ConsensusType {
	case "Raft":
		scale := *n.NodeNet.AllNode
		delete(scale.Member, ID.NodeID{
			ShardID:   0,
			InShardID: 0,
		})
		go n.CycleHeartBeat(intraHeartBeatCycleTime, &scale)
	default:
		go n.CycleHeartBeat(intraHeartBeatCycleTime, n.NodeNet.Shards[n.NodeNet.SelfNodeID.ShardID])
	}

	nodeMsg := Message.NodeMsg{
		TimeStamp:   0,
		From:        n.NodeNet.SelfNodeID,
		To:          ID.NodeID{},
		MsgType:     "IntraLeaderRegistration",
		MsgBody:     nil,
		IntraTerm:   0,
		BlockHeight: 0,
	}

	n.NodeNet.SeedNodeCh <- &nodeMsg

}

func (n *Node) RaftStartMining() {

	newBlock := new(DB.Block)
	newBlock.BlockBody = *n.NodeTxPool.CommitTrs()

	if n.NodeState.BlockHeight == 0 {
		n.Logger.Debug(n.NodeNet.SelfNodeID.String() + " This is First Block")
		newBlock.PreBlockHash = []byte{}
	} else {
		newBlock.PreBlockHash = n.NodeBC.Tip
	}
	n.NodeState.BlockHeight++
	newBlock.BlockHeight = n.NodeState.BlockHeight

	newBlock.TimeStamp = time.Now().UnixNano()
	newBlock.Miner = n.NodeNet.SelfNodeID
	newBlock.SetHash()

	n.NodeBC.AddNewBlock(newBlock)

	msgBody, err := json.Marshal(newBlock)
	check(err)

	nodeMsg := Message.NodeMsg{
		TimeStamp:   0,
		From:        ID.NodeID{},
		To:          ID.NodeID{},
		MsgType:     "block",
		MsgBody:     msgBody,
		IntraTerm:   0,
		BlockHeight: 0,
	}

	scale := *n.NodeNet.AllNode
	delete(scale.Member, ID.NodeID{
		ShardID:   0,
		InShardID: 0,
	})
	n.NodeNet.BroadCastMessage(&scale, &nodeMsg)
	// 复用一下FRaft Q2确认流程
	n.NodeCh.Q2CheckTrigger <- struct{}{}

}

func (n *Node) RaftCheckCommit() {
	Q2 := uint(math.Floor(float64(len(n.NodeNet.AllNode.Member)-1) * 0.5))
	for {
		select {
		case stateMsg := <-n.NodeCh.Q2CheckMsgCh:
			if stateMsg.BlockHeight == n.NodeState.BlockHeight {

			} else {
				continue
			}
			//清空计数器
			if n.NodeState.IntraQ2Count.BlockHeight < n.NodeState.BlockHeight {
				n.NodeState.IntraQ2Count.Count = 0
				n.NodeState.IntraQ2Count.BlockHeight = n.NodeState.BlockHeight
			} else {
				//do something
			}

			if n.NodeState.IntraQ2Count.Count < Q2 {

				n.NodeState.IntraQ2Count.Count++
				if n.NodeState.IntraQ2Count.Count == Q2 {

					info := fmt.Sprintf("%s get enough ConfirmBlockMsg for BlockNo.%d, Q2:%d",
						n.NodeNet.SelfNodeID.String(), n.NodeState.BlockHeight, Q2)
					n.Logger.Debug(info)

					if n.NodeState.BlockHeight == TestingMaxBlockHeight {
						n.Logger.Info(n.NodeNet.SelfNodeID.String() + " stop Mining")

						nodeMsg := Message.NodeMsg{
							TimeStamp:   0,
							From:        n.NodeNet.SelfNodeID,
							To:          ID.NodeID{},
							MsgType:     "stopSendTr",
							MsgBody:     nil,
							IntraTerm:   0,
							BlockHeight: 0,
						}
						n.NodeNet.SeedNodeCh <- &nodeMsg
						return
					} else {
						n.RaftStartMining()
					}
					return
				} else {
					continue
				}

			} else {
				n.Logger.Debug("error in RaftCheckCommit!")
				return
			}
		case <-n.NodeCh.Q2CheckBreak:
			return
		}
	}

}

func (n *Node) SSRaftStartMining() {

	n.NodeState.FRaftBatchBlockCount++

	newBlock := new(DB.Block)
	newBlock.BlockBody = *n.NodeTxPool.CommitTrs()

	if n.NodeState.BlockHeight == 0 {
		n.Logger.Debug(n.NodeNet.SelfNodeID.String() + " This is First Block")
		newBlock.PreBlockHash = []byte{}
	} else {
		newBlock.PreBlockHash = n.NodeBC.Tip
	}
	n.NodeState.BlockHeight++
	newBlock.BlockHeight = n.NodeState.BlockHeight

	newBlock.TimeStamp = time.Now().UnixNano()
	newBlock.Miner = n.NodeNet.SelfNodeID
	newBlock.SetHash()

	n.NodeBC.AddNewBlock(newBlock)

	msgBody, err := json.Marshal(newBlock)
	check(err)

	nodeMsgWithBock := Message.NodeMsg{
		TimeStamp:   0,
		From:        ID.NodeID{},
		To:          ID.NodeID{},
		MsgType:     "block",
		MsgBody:     msgBody,
		IntraTerm:   n.NodeState.IntraTerm,
		BlockHeight: n.NodeState.BlockHeight,
	}
	nodeMsgWithBockHsh := Message.NodeMsg{
		TimeStamp:   0,
		From:        ID.NodeID{},
		To:          ID.NodeID{},
		MsgType:     "blockHash",
		MsgBody:     newBlock.BlockHash,
		IntraTerm:   n.NodeState.IntraTerm,
		BlockHeight: n.NodeState.BlockHeight,
	}

	n.NodeNet.BroadCastMessage(n.NodeNet.Shards[n.NodeNet.SelfNodeID.ShardID], &nodeMsgWithBockHsh)
	n.NodeNet.BroadCastMessage(n.NodeNet.ShardLeaders, &nodeMsgWithBock)

	// 复用一下FRaft Q2确认流程
	n.NodeCh.Q2CheckTrigger <- struct{}{}

}

func (n *Node) SSRaftCheckCommit() {
	Q2 := uint(math.Floor(float64(len(n.NodeNet.Shards[n.NodeNet.SelfNodeID.ShardID].Member)) * 0.5))
	GQ2 := uint(math.Floor(float64(len(n.NodeNet.ShardLeaders.Member)) * 0.5))
	for {
		select {
		case stateMsg := <-n.NodeCh.Q2CheckMsgCh:
			info := fmt.Sprintf("%s get ConfirmBlock from %s, ConfirmHeight: %d, selfHeight: %d",
				n.NodeNet.SelfNodeID.String(),
				stateMsg.From.String(),
				stateMsg.BlockHeight,
				n.NodeState.BlockHeight)
			n.Logger.Debug(info)

			if stateMsg.BlockHeight == n.NodeState.BlockHeight {
				//
			} else {
				continue
			}

			if n.NodeState.IntraQ2Count.BlockHeight < n.NodeState.BlockHeight {
				n.NodeState.IntraQ2Count.Count = 0
				n.NodeState.IntraQ2Count.GlobalCount = 0
				n.NodeState.IntraQ2Count.BlockHeight = n.NodeState.BlockHeight
			} else {

			}

			//如果消息来自片内且片内确认数量未抵达上限
			if stateMsg.From.ShardID == n.NodeNet.SelfNodeID.ShardID &&
				n.NodeState.IntraQ2Count.Count < Q2 {
				//计数片内确认消息
				n.NodeState.IntraQ2Count.Count++
			}
			//如果消息来自其他分片且确认数量未抵达上限
			if stateMsg.From.ShardID != n.NodeNet.SelfNodeID.ShardID &&
				n.NodeState.IntraQ2Count.GlobalCount < GQ2 {
				//计数片间确认消息，更新最近一次收到的片间确认消息的发送者
				n.NodeState.IntraQ2Count.GlobalCount++
				n.NodeState.IntraQ2Count.LatestResponseID = stateMsg.From
			}
			//如果片内片间确认消息抵达上限
			if n.NodeState.IntraQ2Count.Count == Q2 &&
				n.NodeState.IntraQ2Count.GlobalCount == GQ2 {
				info1 := fmt.Sprintf("%s get enough ConfirmBlockMsg fro Block No.%d, Q2:%d, GQ2:%d",
					n.NodeNet.SelfNodeID.String(), n.NodeState.BlockHeight, Q2, GQ2)
				n.Logger.Debug(info1)

				if n.NodeState.BlockHeight == TestingMaxBlockHeight {
					n.Logger.Info(n.NodeNet.SelfNodeID.String() + " stop Mining")

					nodeMsg := Message.NodeMsg{
						TimeStamp:   0,
						From:        n.NodeNet.SelfNodeID,
						To:          ID.NodeID{},
						MsgType:     "stopSendTr",
						MsgBody:     nil,
						IntraTerm:   0,
						BlockHeight: 0,
					}
					n.NodeNet.SeedNodeCh <- &nodeMsg
					return
				} else {
					//一个ShardLeader的出块任期结束
					if n.NodeState.FRaftBatchBlockCount == BatchBlocksNum {

						n.NodeState.FRaftBatchBlockCount = 0
						//n.SSRaftNextMiner()
						return
					} else {
						n.SSRaftStartMining()
						return
					}

				}

			} else {
				continue
			}
		case <-n.NodeCh.Q2CheckBreak:
			return
		}
	}

}

func (n *Node) SSRaftNextMiner() {

	nodeMsg := Message.NodeMsg{
		TimeStamp:   0,
		From:        n.NodeNet.SelfNodeID,
		To:          ID.NodeID{},
		MsgType:     "StartCommit",
		MsgBody:     n.NodeBC.Tip,
		IntraTerm:   0,
		BlockHeight: 0,
	}

	//n.NodeNet.SendMessage(n.NodeState.IntraQ2Count.LatestResponseID, &nodeMsg)
	n.NodeNet.AllNode.Member[n.NodeState.IntraQ2Count.LatestResponseID] <- &nodeMsg
}

///*************************************/
//
//func (n *Node) FRaftStartMiningV1(lastHash []byte) {
//
//	n.NodeState.FRaftBatchBlockCount++
//
//	newBlock := new(DB.Block)
//	newBlock.BlockBody = *n.NodeTxPool.CommitTrs()
//
//	if n.NodeID.ShardID == 1 &&
//		n.NodeState.BlockHeight <= BatchBlocksNum {
//
//		if n.NodeState.BlockHeight == 0 {
//			n.Logger.Debug(n.NodeID.String() + " This is First Block")
//			newBlock.PreBlockHash = []byte{}
//		} else {
//			newBlock.PreBlockHash = n.NodeBC.Tip
//		}
//		n.NodeState.BlockHeight++
//		newBlock.BlockHeight = n.NodeState.BlockHeight
//
//		newBlock.TimeStamp = time.Now().UnixNano()
//		newBlock.Miner = n.NodeID
//		newBlock.SetHash()
//
//		n.NodeBC.AddNewBlock(newBlock)
//
//		msgBody, err := json.Marshal(newBlock)
//		check(err)
//		n.BroadCastMessage("intra", "block", msgBody)
//		n.BroadCastMessage("global", "block", msgBody)
//		// 复用一下FRaft Q2确认流程
//		n.NodeCh.Q2CheckTrigger <- struct{}{}
//
//	}
//
//}
