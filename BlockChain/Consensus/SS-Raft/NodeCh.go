package Node

import (
	"BlockChain/Message"
)

type ChSum struct {
	// 优雅退出管道
	ElegantExitCh chan struct{}

	// 心跳停止管道
	IntraStopHeartCh chan struct{}

	Q1CheckTrigger chan struct{}
	Q1CheckBreak   chan struct{}
	Q1CheckMsgCh   chan *Message.NodeMsg

	Q2CheckTrigger chan struct{}
	Q2CheckBreak   chan struct{}
	Q2CheckMsgCh   chan *Message.NodeMsg
}

func NewChSum(exitCh chan struct{}) *ChSum {

	value := ChSum{
		ElegantExitCh:    exitCh,
		IntraStopHeartCh: make(chan struct{}, 1),

		Q1CheckTrigger: make(chan struct{}, 1),
		Q1CheckBreak:   make(chan struct{}, 1),
		Q1CheckMsgCh:   make(chan *Message.NodeMsg, 128),

		Q2CheckTrigger: make(chan struct{}, 1),
		Q2CheckBreak:   make(chan struct{}, 1),
		Q2CheckMsgCh:   make(chan *Message.NodeMsg, 128),
	}
	return &value
}
