package Message

import (
	"BlockChain/DB"
	"encoding/json"
	"log"
	"strings"
	"time"
)

type NodeMsg struct {
	TimeStamp int64

	From    string
	To      string
	MsgType string
	MsgBody []byte

	EpochNum     int
	BlockHeight  uint64
	ConsensusTye string
}

func check(err error) {

	if err != nil {
		log.Panic(err)
	}

}

// NewNodeMsg 传入的参数中，msgBody为原始数据，本函数会将msgBody转换为[]byte
func NewNodeMsg(from, to, msgType string, msgBody interface{}) *NodeMsg {

	marshalData, err := json.Marshal(msgBody)
	check(err)

	returnValue := NodeMsg{
		TimeStamp:    time.Now().Unix(),
		From:         from,
		To:           to,
		MsgType:      msgType,
		MsgBody:      marshalData,
		EpochNum:     0,
		BlockHeight:  0,
		ConsensusTye: "",
	}

	return &returnValue
}

// Marshal 消息序列化为数据流
func (nm *NodeMsg) Marshal() []byte {

	marshalData, err := json.Marshal(nm)
	check(err)

	return marshalData
}

// UnmarshalBlock 解析为区块数据
func (nm *NodeMsg) UnmarshalBlock() *DB.Block {

	data := new(DB.Block)
	err := json.Unmarshal(nm.MsgBody, data)
	check(err)

	return data
}

func (nm *NodeMsg) UnmarshalShardMem() *map[string]string {

	data := new(map[string]string)
	err := json.Unmarshal(nm.MsgBody, data)
	check(err)

	return data
}

func (nm *NodeMsg) UnmarshalFindNode() *map[int]int {

	data := new(map[int]int)
	err := json.Unmarshal(nm.MsgBody, data)
	check(err)

	return data
}

func (nm *NodeMsg) UnmarshalClientAddr() string {

	return strings.Trim(string(nm.MsgBody), "\"")

}
