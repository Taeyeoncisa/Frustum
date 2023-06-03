package SS_PBFT_V1

import "BlockChain/Message"

type ChSum struct {
	ElegantExitCh chan struct{}
	CLI           chan string
	SendMsgCh     chan *Message.NodeMsg
	ReceiveMsgCh  chan *Message.NodeMsg
}

func NewChSum() *ChSum {
	return &ChSum{
		ElegantExitCh: make(chan struct{}, 1),
		CLI:           make(chan string, 16),
		SendMsgCh:     make(chan *Message.NodeMsg, 1024),
		ReceiveMsgCh:  make(chan *Message.NodeMsg, 1024),
	}
}
