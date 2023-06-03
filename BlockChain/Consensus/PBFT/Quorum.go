package PBFT

import (
	"BlockChain/Message"
	"math"
	"sync"
)

type Quorum struct {
	Name              string
	LatestProposalNum uint64
	VoteCount         int
	Finish            bool

	MsgBuffer []Message.NodeMsg
	//MsgBuffer互斥锁，可能存在清空buffer时，收到消息再往里面写
	flag sync.Mutex
}

func NewQuorum(str string) *Quorum {

	return &Quorum{
		Name:              str,
		LatestProposalNum: 0,
		VoteCount:         0,
		Finish:            false,
		MsgBuffer:         make([]Message.NodeMsg, 0),
	}

}

// ResetCount 计票器清零，轮次更新到最新
func (q *Quorum) ResetCount(proposalNum uint64) {

	q.flag.Lock()
	defer q.flag.Unlock()

	q.LatestProposalNum = proposalNum
	q.VoteCount = 0
	q.Finish = false

	if len(q.MsgBuffer) == 0 {
		//无历史消息
	} else {
		newMsgBuffer := make([]Message.NodeMsg, 0)
		//历史消息buffer中轮次大于等于节点最新
		for _, msg := range q.MsgBuffer {

			if msg.BlockHeight >= proposalNum {
				newMsgBuffer = append(newMsgBuffer, msg)
			}

		}

		//仅保留高于本节点轮次的消息
		q.MsgBuffer = q.MsgBuffer[:0]
		q.MsgBuffer = newMsgBuffer

	}
}

// Handle 处理msgBuffer中的消息，f为恶意节点占比，totalNum为共识成员总数，msg待处理消息
// 返回true, XX代表达成法定人数；返回false, XX代表未达成法定人数
// 返回false, 1024代表已经达成法定人数，不再计票；
// 返回false, 1025代表消息轮次高于本节点最高轮次；
func (q *Quorum) Handle(f float64, totalNum int, msg *Message.NodeMsg) (bool, int) {

	//恶意节点数量
	MaliciousNodes := int(math.Floor(float64(totalNum) * f))
	//达成共识的法定节点数量
	var QuorumNodes int

	switch q.Name {
	case "IntraFinality", "GlobalFinality", "Finality":
		//对应客户端只需f+1回复即可确认提交成功
		QuorumNodes = MaliciousNodes + 1
	default:
		QuorumNodes = 2*MaliciousNodes + 1
	}
	//fmt.Printf("MaliciousNum: %d, f: %f, TotalNum: %d,  %s quorum: %d\n", MaliciousNodes, f, totalNum, q.Name, QuorumNodes)

	//只有处理本轮消息
	if msg.BlockHeight == q.LatestProposalNum {

		if q.VoteCount < QuorumNodes {

			q.VoteCount++
			if q.VoteCount == QuorumNodes {
				return true, QuorumNodes
			} else {
				//若未达成法定人数，再查看缓存
				for _, v := range q.MsgBuffer {

					if v.BlockHeight == q.LatestProposalNum {
						q.VoteCount++
						if q.VoteCount == QuorumNodes {
							//缓存消息满足达成共识
							return true, QuorumNodes
						}
					} else {
						continue
					}

				}
				//查询完缓存仍未达成共识
				return false, QuorumNodes

			}

		} else {
			//本轮法定人数已达成
			return false, 1024
		}

	} else if msg.BlockHeight > q.LatestProposalNum {
		//如果消息轮次高于节点自身最新轮次则写入缓存buffer
		//可能存在并发的清空操作，添加互斥锁
		q.flag.Lock()
		q.MsgBuffer = append(q.MsgBuffer, *msg)
		q.flag.Unlock()

		//消息轮次高于节点最高轮次
		return false, 1025
	} else {
		//历史消息则丢弃不处理
		return false, QuorumNodes
	}

}
