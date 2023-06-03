package TxPool

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"math/rand"
	"time"
)

type Transaction struct {
	TimeStamp int64
	Level     int
	From      int
	To        int
	Data      []byte
	Hash      [32]byte
	OrigHash  [32]byte
}

// NewTransaction 随机生成新交易，默认三个等级，比例为1：2：7
func NewTransaction(shardNum int) *Transaction {

	rand.Seed(time.Now().UnixNano())

	tr := new(Transaction)
	tr.From = rand.Intn(shardNum)
	tr.To = rand.Intn(shardNum)

	tr.TimeStamp = time.Now().UnixNano()

	tr.Hash = sha256.Sum256(tr.Serialize())

	return tr
}

// Serialize 交易结构体序列化为[]byte
func (tr *Transaction) Serialize() []byte {

	result := new(bytes.Buffer)
	encoder := gob.NewEncoder(result)

	err := encoder.Encode(tr)
	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}
