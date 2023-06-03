package SeedNode

import (
	"BlockChain/CLI"
	"BlockChain/ID"
	"BlockChain/Log"
	"BlockChain/Message"
	"BlockChain/NetWork"
	"encoding/csv"
	"fmt"
	"go.uber.org/zap"
	"io"
	"log"
	"os"
	"time"
)

const timeFormat = "2006-01-02 15:04:05.000"

type SeedNode struct {
	Logger        *zap.SugaredLogger
	SeedNodeCh    *ChSum
	SeedNodeState *State
	Peers         *NetWork.NodeTable

	NetWork NetWork.NetWork
}

func NewSeedNode(logger *zap.SugaredLogger, nodeID string) *SeedNode {
	chs := NewChSum()
	server := NetWork.NewServerV2(
		logger,
		NetWork.GetNatIP()+":"+"10000",
		nil,
		chs.ElegantExitCh,
		chs.ReceiveMsgCh,
	)

	node := SeedNode{
		Logger:        logger,
		SeedNodeCh:    chs,
		SeedNodeState: NewState(),
		Peers:         NetWork.NewNodeTable(nodeID),
		NetWork:       server,
	}

	return &node
}

func (n *SeedNode) Run() {

	go n.NetWork.Start()
	go CLI.Input(n.SeedNodeCh.CLI)

	for {
		select {
		case <-n.SeedNodeCh.ElegantExitCh:
			//n.NodeNet.ElegantExitCh <- struct{}{}
			n.Logger.Debug("SeedNode gracefully shutdown")
			return
		case cli := <-n.SeedNodeCh.CLI:
			n.CLIHandle(cli)
		case msg := <-n.SeedNodeCh.ReceiveMsgCh:
			n.MSGHandle(msg)
		case msg := <-n.SeedNodeCh.SendMsgCh:
			go n.SendMsg(msg)
		}
	}

}

func (n *SeedNode) CLIHandle(cli string) {

	switch cli {

	case "start":

		switch n.SeedNodeState.ConsensusType {

		case "RapidChain":
			n.GiveStartToIntraLeaders()

		default:
			n.GiveStartToGlobalLeader()
		}

	case "selectGlobalLeader":
		n.SelectGlobalLeader()
		n.SendGlobalLeaderMem()

		go func() {
			time.Sleep(2 * time.Second)
			n.SeedNodeCh.CLI <- "start"
		}()

	case "selectIntraLeader":
		n.SelectIntraLeader()
		n.SendIntraLeaderMem()
		go func() {
			time.Sleep(2 * time.Second)
			n.SeedNodeCh.CLI <- "selectGlobalLeader"
		}()

	case "sendMem":
		n.SendMem()
		go func() {
			time.Sleep(2 * time.Second)
			n.SeedNodeCh.CLI <- "selectIntraLeader"
		}()

	case "caTx":
		n.ReadETHTx()

	case "quit":
		n.NetWork.ShutDown()

	case "exit":

		//目标节点ID未定，设置为空
		msg := Message.NewNodeMsg(n.Peers.SelfNodeID, "",
			"ShutDown", nil)

		n.BroadCast("all", 0, msg)
		time.Sleep(5 * time.Second)
		//n.SeedNodeCh.ElegantExitCh <- struct{}{}
		//close(n.NodeNet.ElegantExitCh)
		n.NetWork.ShutDown()
	}

}

// BroadCast scope代表消息广播的消息范围，all代表全部节点，shard参数无效直接输入为0；
// shard代表仅在某一分片内广播，1024为预留特殊编号
func (n *SeedNode) BroadCast(scope string, shard int, msg *Message.NodeMsg) {

	switch scope {

	case "all":

		for ShardID, SHARD := range n.Peers.Shards {

			switch ShardID {

			case 0, 1024, 1025:
				//0，1024为特殊分片ID
			default:
				for k := range SHARD.Member {

					if k == n.Peers.SelfNodeID {
						//自己就不发送
					} else {
						newMsg := *msg
						newMsg.To = k

						n.SeedNodeCh.SendMsgCh <- &newMsg
					}

				}
			}

		}

	case "shard":

		nodes := n.Peers.Shards[shard]

		for k := range nodes.Member {

			if k == n.Peers.SelfNodeID {
				//自己就不发送
			} else {
				newMsg := *msg
				newMsg.To = k

				n.SeedNodeCh.SendMsgCh <- &newMsg
			}

		}

	}

}

// SendMem 将对应的分片成员集合发送至对应的所有节点
func (n *SeedNode) SendMem() {

	for ShardID, SHARD := range n.Peers.Shards {

		switch ShardID {
		case 0, 1024, 1025:

		default:
			for k := range SHARD.Member {

				msg := Message.NewNodeMsg(n.Peers.SelfNodeID, k, "NodeMember", &SHARD.Member)
				n.SeedNodeCh.SendMsgCh <- msg
			}

		}

	}

}

// SendIntraLeaderMem 告知所有节点intraLeader集合
func (n *SeedNode) SendIntraLeaderMem() {

	SHARD := n.Peers.Shards[1024]

	for ShardID, Shard := range n.Peers.Shards {
		switch ShardID {
		case 0, 1024, 1025:

		default:
			for k := range Shard.Member {
				msg := Message.NewNodeMsg(n.Peers.SelfNodeID, k, "IntraLeaderMem", &SHARD.Member)
				n.SeedNodeCh.SendMsgCh <- msg
			}
		}
	}

}

// SelectIntraLeader 分片内选举leader
func (n *SeedNode) SelectIntraLeader() {

	LeaderShard := NetWork.NewShard()
	n.Peers.Shards[1024] = LeaderShard

	for ShardID, SHARD := range n.Peers.Shards {
		switch ShardID {

		case 0, 1024, 1025:

		default:
			intraShardNodeNum := len(SHARD.Member)

			randomLeader := Log.Random(1, intraShardNodeNum)
			for k, v := range SHARD.Member {

				nodeID := ID.StringToID(k)

				if nodeID.InShardID == randomLeader {
					LeaderShard.AddNode(k, v)
					n.Logger.Debugf("The Leader of Shard %d is: %s", nodeID.ShardID, k)
				} else {
					//not leader
				}
			}

		}

	}

}

// SelectGlobalLeader 多个intraLeader组成的commit选举GlobalLeader
func (n *SeedNode) SelectGlobalLeader() {

	LeaderShard := NetWork.NewShard()
	n.Peers.Shards[1025] = LeaderShard

	intraLeaderNum := len(n.Peers.Shards[1024].Member)

	randomLeader := Log.Random(1, intraLeaderNum)
	//fmt.Println(randomLeader, n.NodeNet.Peers.Shards[1024].Member)
	for k, v := range n.Peers.Shards[1024].Member {

		nodeID := ID.StringToID(k)

		if nodeID.ShardID == randomLeader {
			LeaderShard.AddNode(k, v)
			n.Logger.Debugf("The GlobalLeader is: %s", k)
		} else {
			//not leader
		}
	}

}

// SendGlobalLeaderMem 告知intraLeader集合GlobalLeader地址
func (n *SeedNode) SendGlobalLeaderMem() {

	SHARD := n.Peers.Shards[1025]
	for ShardID, Shard := range n.Peers.Shards {
		switch ShardID {
		case 0, 1024, 1025:

		default:
			for k := range Shard.Member {
				msg := Message.NewNodeMsg(n.Peers.SelfNodeID, k, "GlobalLeaderMem", &SHARD.Member)
				n.SeedNodeCh.SendMsgCh <- msg
			}
		}
	}

}

func (n *SeedNode) ReadETHTx() {
	path := "JustFromAndTo.csv"

	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	csvReader := csv.NewReader(file)
	startTime := time.Now()
	n.Logger.Debugf("Start Read File")

	TxTotalNum := 0
	done := make(chan struct{}, 1)
	go func(num int, done chan struct{}) {

		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				log.Printf("TxNum: %d", TxTotalNum)
			case <-done:
				return
			}
		}

	}(TxTotalNum, done)

	for {

		_, errLine := csvReader.Read()
		if errLine == io.EOF {
			break
		}
		if errLine != nil {
			fmt.Println("error in reading csv file")
			log.Fatal(errLine)
		}

		TxTotalNum++

	}

	n.Logger.Debugf("Time cost = %v, TxTotalNum: %d", time.Since(startTime), TxTotalNum)
	close(done)
}

func (n *SeedNode) SendMsg(msg *Message.NodeMsg) {

	ok, targetAddr := n.Peers.FindNode(msg.To)
	if ok {

		err := n.NetWork.SendMsg(targetAddr, msg)
		if err != nil {
			n.Logger.Warnf("Fail send %s to %s", msg.MsgType, msg.To)
		}

	} else {
		n.Logger.Infof("TargetNode %s not exist", msg.To)
	}

}
