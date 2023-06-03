package main

import (
	"crypto/sha256"
	"encoding/json"
)

// 节点信息
type Node struct {
	IP              string
	committeeID     int
	committeeList   map[int][]Node
	commiteeMembers []Node
	isHonest        bool
	prepareNum      int
	commitNum       int
	Sig             bool // 是否签名，用于最后统计成员签名数
	blockChain      Blockchain
}

// 交易信息
type Transaction struct {
	Hash [32]byte
	// 用户ID
	Inputs  int
	Outputs int
	Value   int
}

func (t *Transaction) setHash() {
	t.Hash = t.calculateHash()
}

func (t *Transaction) calculateHash() [32]byte {
	jsonT, _ := json.Marshal(t)
	return sha256.Sum256(jsonT)
}

// 区块信息

// 提案给出的block
type Blockchain []FinalBlock

type ProposedBlock struct {
	PreviousBlockHash [32]byte
	Iteration         uint
	LeaderIP          string
	Transactions      []Transaction // not hashed because it is implicitly in MerkleRoot
}

type FinalBlock struct {
	Hash          [32]byte
	ProposedBlock ProposedBlock
}

func (b *ProposedBlock) setHash() [32]byte {
	jsonB, _ := json.Marshal(b)
	return sha256.Sum256(jsonB)
}
