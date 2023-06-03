package SS_PBFT_V2

import (
	"BlockChain/Message"
)

type ChSum struct {
	MainProcessExit chan struct{}
	CLI             chan string
	SenMsgCh        chan *Message.NodeMsg
	RecMsgCh        chan *Message.NodeMsg
	TxWorkerStop    chan struct{}
}

func NewChSum() *ChSum {

	return &ChSum{
		MainProcessExit: make(chan struct{}, 1),
		CLI:             make(chan string, 16),
		SenMsgCh:        make(chan *Message.NodeMsg, 2048),
		RecMsgCh:        make(chan *Message.NodeMsg, 2048),
		TxWorkerStop:    make(chan struct{}, 1),
	}

}
