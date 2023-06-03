package Consensus

import "BlockChain/Message"

type Node interface {
	Run()
	CLIHandle(cli string)
	MSGHandle(msg *Message.NodeMsg)
}
