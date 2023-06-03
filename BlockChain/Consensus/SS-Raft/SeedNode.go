package Node

import (
	"SS-Raft/ID"
	"SS-Raft/Message"
	"SS-Raft/NetWork"
	"SS-Raft/TxPool"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"strconv"
	"time"
)

type SeedNode struct {
	Logger      *zap.Logger
	SeedNodeNet *NetWork.NetWork

	ElegantExitCh    chan struct{}
	ShutDownSendTrCh chan struct{}
	NodesTxPool      map[ID.NodeID]*TxPool.TxPool
	SendTrTPS        int
	SendTrWorkerNum  int
	ConsensusType    string
	CliCh            chan *string
}

func NewSeedNode(logger *zap.Logger,
	done chan struct{},
	sendTPS int,
	consensus string,
	cliCh chan *string) *SeedNode {
	return &SeedNode{
		Logger:           logger,
		SeedNodeNet:      NetWork.NewNetWork("0-0"),
		ElegantExitCh:    done,
		ShutDownSendTrCh: make(chan struct{}, 1),
		NodesTxPool:      nil, //main函数中直接赋值获得
		SendTrTPS:        sendTPS,
		SendTrWorkerNum:  0,
		ConsensusType:    consensus,
		CliCh:            cliCh,
	}
}

func (sn *SeedNode) UpdateNodeNet(chs *map[ID.NodeID]chan *Message.NodeMsg) {
	for k, v := range *chs {
		//同步所有节点的消息接收管道
		sn.SeedNodeNet.AllNode.AddMem(k, v)
		//同步分片消息管道
		if _, ok := sn.SeedNodeNet.Shards[k.ShardID]; ok {
			sn.SeedNodeNet.Shards[k.ShardID].AddMem(k, v)
		} else {
			sn.SeedNodeNet.Shards[k.ShardID] = NetWork.NewMemberManager()
			sn.SeedNodeNet.Shards[k.ShardID].AddMem(k, v)
		}
	}
	//同步自身消息接收管道
	sn.SeedNodeNet.SelfRecCh = sn.SeedNodeNet.AllNode.Member[sn.SeedNodeNet.SelfNodeID]
	//剔除shards中seedNode的分片
	delete(sn.SeedNodeNet.Shards, 0)
}

func (sn *SeedNode) Run() {
	go sn.SeedNodeNet.Run(sn.ElegantExitCh)
	for {
		select {
		case <-sn.ElegantExitCh:
			sn.Logger.Info("SeedNode Exit!")
			return
		case value := <-sn.CliCh:
			sn.HandleCLI(value)
		case msg := <-sn.SeedNodeNet.SelfRecCh:
			sn.HandleMsg(msg)
		}
	}
}

func (sn *SeedNode) HandleCLI(str *string) {
	switch *str {
	case "/quit":
		close(sn.ElegantExitCh)
		time.Sleep(2 * time.Second)
	case "/HelloWorld":
		nodeMsg := Message.NodeMsg{
			TimeStamp:   0,
			From:        ID.NodeID{},
			To:          ID.NodeID{},
			MsgType:     "HelloWorld",
			MsgBody:     nil,
			IntraTerm:   0,
			BlockHeight: 0,
		}
		sn.SeedNodeNet.BroadCastMessage(sn.SeedNodeNet.AllNode, &nodeMsg)
	case "/start":
		nodeMsg := Message.NodeMsg{
			TimeStamp:   0,
			From:        ID.NodeID{},
			To:          ID.NodeID{},
			MsgType:     "start",
			MsgBody:     nil,
			IntraTerm:   0,
			BlockHeight: 0,
		}
		sn.SeedNodeNet.BroadCastMessage(sn.SeedNodeNet.AllNode, &nodeMsg)
	case "/sendTr":
		sn.SendTr()
	case "/analysis":
		nodeMsg := Message.NodeMsg{
			TimeStamp:   0,
			From:        ID.NodeID{},
			To:          ID.NodeID{},
			MsgType:     "analysis",
			MsgBody:     nil,
			IntraTerm:   0,
			BlockHeight: 0,
		}
		switch sn.ConsensusType {
		case "Raft":
			for _, v := range sn.SeedNodeNet.ShardLeaders.Member {
				v <- &nodeMsg
			}
		default:
			for k, v := range sn.SeedNodeNet.ShardLeaders.Member {
				if k.ShardID == 1 {
					v <- &nodeMsg
				} else {
					// do something
				}
			}

		}

	case "/readTest":

	case "/consensus":
		fmt.Printf("Consensus is: %s\n", sn.ConsensusType)
		fmt.Printf("%d Shards, %d ShardLeaders\n",
			len(sn.SeedNodeNet.Shards),
			len(sn.SeedNodeNet.ShardLeaders.Member))

	case "/member":
		//fmt.Println(*sn.SeedNodeNet.AllNode)
		for k1, v1 := range sn.SeedNodeNet.Shards {
			var members []string
			for k := range v1.Member {
				members = append(members, k.String())
			}
			fmt.Printf("Shard No.%s %s\n", strconv.Itoa(k1), members)
		}
		for k2 := range sn.SeedNodeNet.ShardLeaders.Member {
			fmt.Printf("ShardLeader %s\n", k2.String())
		}

	default:
		fmt.Printf("Don't Know this Command, Please Read the Code!\n")
	}
}

func (sn *SeedNode) HandleMsg(msg *Message.NodeMsg) {
	switch msg.MsgType {
	case "IntraLeaderRegistration":
		info := fmt.Sprintf("%s Get IntraLeaderRegistration from %s",
			sn.SeedNodeNet.SelfNodeID.String(),
			msg.From.String())
		sn.Logger.Info(info)
		//从已初始化的所有管道信息提取对应的管道至ShardLeaders
		sn.SeedNodeNet.ShardLeaders.Member[msg.From] = sn.SeedNodeNet.AllNode.Member[msg.From]
		//根据不同的共识协议，同步leader地址至其他节点
		if sn.ConsensusType == "Raft" {
			//开始同步leader地址给其他节点
		} else {
			if len(sn.SeedNodeNet.Shards) == len(sn.SeedNodeNet.ShardLeaders.Member) {
				//所有分片leader选举完成，开始同步leader地址给其他节点
			} else {
				return
			}
		}
		//准备leaders地址信息
		var leaders []ID.NodeID
		for k := range sn.SeedNodeNet.ShardLeaders.Member {
			leaders = append(leaders, k)
		}
		leadersData, _ := json.Marshal(leaders)
		nodeMsg := Message.NodeMsg{
			TimeStamp:   0,
			From:        ID.NodeID{},
			To:          ID.NodeID{},
			MsgType:     "GlobalMem",
			MsgBody:     leadersData,
			IntraTerm:   0,
			BlockHeight: 0,
		}
		sn.SeedNodeNet.BroadCastMessage(sn.SeedNodeNet.AllNode, &nodeMsg)
		go sn.SendTr()

	case "stopSendTr":
		close(sn.ShutDownSendTrCh)
		select {
		case <-time.After(3 * time.Second):
			str := "/analysis"
			sn.CliCh <- &str
		}
	}
}

func (sn *SeedNode) SendTrWorker(receiveCh chan *Message.NodeMsg) {
	ticker := time.NewTicker(10 * time.Millisecond)
	for {

		select {
		case <-ticker.C:
			msgBody, err := json.Marshal(TxPool.NewTransaction())
			check(err)
			nodeMsg := Message.NodeMsg{
				TimeStamp:   time.Now().UnixNano(),
				From:        sn.SeedNodeNet.SelfNodeID,
				To:          ID.NodeID{},
				MsgType:     "Transaction",
				MsgBody:     msgBody,
				IntraTerm:   0,
				BlockHeight: 0,
			}
			receiveCh <- &nodeMsg

		case <-sn.ShutDownSendTrCh:
			ticker.Stop()
			return
		}

	}
}

func (sn *SeedNode) SendStartCommit(ch chan *Message.NodeMsg) {
	select {

	case <-time.After(500 * time.Millisecond):
		nodeMsg := Message.NodeMsg{
			TimeStamp:   0,
			From:        sn.SeedNodeNet.SelfNodeID,
			To:          ID.NodeID{},
			MsgType:     "StartCommit",
			MsgBody:     nil,
			IntraTerm:   0,
			BlockHeight: 0,
		}

		ch <- &nodeMsg

	}
}

func (sn *SeedNode) SendTr() {

	select {
	case <-time.After(3 * time.Second):

	}

	sn.SendTrWorkerNum = sn.SendTrTPS / 100
	for k, v := range sn.SeedNodeNet.ShardLeaders.Member {
		//每个worker的TPS为100，根据需求TPS推算需要worker数量，然后并发
		switch sn.ConsensusType {
		case "Raft":
			sn.SendStartCommit(v)
			for i := 0; i < sn.SendTrWorkerNum; i++ {
				go sn.SendTrWorker(v)
			}
		case "SS-Raft":
			if k.ShardID == 1 {
				sn.SendStartCommit(v)
				for i := 0; i < sn.SendTrWorkerNum; i++ {
					go sn.SendTrWorker(v)
				}
			} else {
				//仅模拟了一个分区在挖矿
				//for i := 0; i < sn.SendTrWorkerNum; i++ {
				//	go sn.SendTrWorker(v)
				//}
			}
		case "SS-Raft-Pro":

		}
	}
}
