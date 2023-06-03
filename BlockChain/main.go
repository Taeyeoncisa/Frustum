package main

import (
	"BlockChain/Consensus"
	"BlockChain/Consensus/RapidChain"
	"BlockChain/ID"
	"BlockChain/Log"
	"BlockChain/SeedNode"
	"github.com/spf13/pflag"
)

var (
	consensusType = pflag.IntP("consensusType",
		"c",
		0,
		"which consensus the node use, 0:xxx, 1:xxx ")
	txArriveTPS = pflag.IntP("txArriveTPS",
		"t",
		100,
		"the number of txs arrive in node per second")
	loggerLevel = pflag.IntP("loggerLevel",
		"l",
		0,
		"the log level in node, 0: info, 1: debug")
	nodeID = pflag.StringP("nodeID",
		"n",
		"0-0",
		"the node's ID in the network")
	port = pflag.StringP("port",
		"p",
		"10000",
		"the node's port")
)

func main() {

	/*******************************************************************/
	pflag.Parse()

	/*******************************************************************/
	//必须放在参数解析之后，否则出现所有节点ID为0-0
	NodeID := ID.StringToID(*nodeID)
	/*******************************************************************/
	//定制logger
	logger := Log.LoggerInit(*loggerLevel, NodeID)
	defer logger.Sync()
	/*******************************************************************/

	var node Consensus.Node

	/*******************************************************************/
	switch NodeID.ShardID {
	//种子节点
	case 0:
		node = SeedNode.NewSeedNode(logger, *nodeID)
	//共识节点
	default:
		switch *consensusType {

		case 0:
			//node := SS_PBFT_V2.NewNode(logger, *nodeID, *ipAddr)
			//node := SS_PBFT_V1.NewNode(logger, NodeID, *ipAddr)
			node = RapidChain.NewNode(logger, *nodeID, *port)
		case 1:
			//node := Elastico.NewNode(logger, NodeID, *ipAddr)
		}

	}
	node.Run()
	/*******************************************************************/

}
