package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

// elastico加了身份验证的过程
// final commitee

var flagArgs FlagArgs
var txList []Transaction
var nodes []Node

// 全局区块链
var globalblockChain Blockchain

// 初始化用户
func initNodes() {
	nodes = append(nodes, Node{})
	for i := 1; i <= int(flagArgs.n); i++ {
		var node Node = Node{
			IP:          "0,0,0,0",
			committeeID: -1,
			Sig:         false,
			isHonest:    true,
			prepareNum:  0,
			commitNum:   0,
			blockChain:  globalblockChain,
		}
		nodes = append(nodes, node)
	}
	faultyNum := flagArgs.n / 3
	rand.Seed(time.Now().UnixNano())
	for i := 1; i <= int(faultyNum); i++ {
		randomN := rand.Intn(int(flagArgs.n)-1) + 1
		nodes[randomN].isHonest = false
	}
}
func verifyBlock(block ProposedBlock) {
	for _ = range block.Transactions {

	}
}

// 生成交易
func txGen() {
	// 随机生成付款方和收款方及交易金额
	for i := 0; i < int(flagArgs.txNum); i++ {
		t := Transaction{}
		t.Inputs = rand.Intn((int(flagArgs.n) - 1) + 1)
		t.Outputs = rand.Intn((int(flagArgs.n) - 1) + 1)
		t.Value = rand.Intn((int(flagArgs.n) - 1) + 1)
		t.setHash()
		txList = append(txList, t)
	}
}

// 从tx中获得交易列表
func getTxList(wg *sync.WaitGroup, m *sync.Mutex) []Transaction {
	m.Lock()
	if len(txList) < int(flagArgs.txPerBlock) {
		return txList
	}
	txs := txList[:flagArgs.txPerBlock]
	txList = txList[flagArgs.txPerBlock:]
	m.Unlock()
	wg.Done()
	return txs
}
func receivePre(nodes []Node, block ProposedBlock, ID int, finalCommit chan bool) {
	// pre-prepare
	// 验证交易
	verifyBlock(block)
	rand.Seed(time.Now().UnixNano())
	if !nodes[ID].isHonest && rand.Intn(100) <= 50 {
		return
	}
	// 进入prepare阶段
	// 1.向节点发送prepare信息
	for i := 0; i < int(flagArgs.n_per_commitee); i++ {
		nodes[i].prepareNum += 1
	}
	// 2.等待prepare消息
	for nodes[ID].prepareNum <= int(2*flagArgs.totalF) {
	}
	// 进入commit阶段
	verifyBlock(block)
	// 1.向节点发送commit信息
	for i := 0; i < int(flagArgs.n_per_commitee); i++ {
		nodes[i].commitNum += 1
	}
	// 2.等待commit消息
	for nodes[ID].commitNum <= int(2*flagArgs.totalF) {
	}
	// 最终提交
	nodes[ID].Sig = true
	finalCommit <- true
}
func Elastico() {
	// 当前阶段状态信息
	startTime := time.Now()
	nodes_iter := nodes
	var committeeList map[int][]Node = make(map[int][]Node)
	for i := 1; i <= int(flagArgs.n); i++ {
		nodes_iter[i].prepareNum = 0
		nodes_iter[i].commitNum = 0
		nodes_iter[i].Sig = false
	}
	for i := 0; i < int(flagArgs.commitNum); i++ {
		committeeList[i] = make([]Node, 0)
	}
	// 统计划分委员会的时间
	// 1.划分委员会
	for i := 1; i <= int(flagArgs.n); i++ {
		committeeList[i%int(flagArgs.commitNum)] = append(committeeList[i%int(flagArgs.commitNum)], nodes_iter[i])
		nodes_iter[i].committeeID = i % int(flagArgs.commitNum)
	}
	// 把委员会信息广播给每个节点
	for i := 1; i <= int(flagArgs.n); i++ {
		nodes_iter[i].committeeList = committeeList
		nodes_iter[i].commiteeMembers = committeeList[nodes_iter[i].committeeID]
	}
	fmt.Println("执行时间：", time.Since(startTime))
	// 2.m个委员会各自运行
	runPBFT := make(chan bool, 1000)
	for i := 0; i < int(flagArgs.commitNum); i++ {
		go PBFT(nodes_iter, committeeList[i], runPBFT)
	}
	// TODO：管道阻塞
	for i := 0; i < int(flagArgs.commitNum); i++ {
		<-runPBFT
	}
}
func addBlock(block FinalBlock, wg *sync.WaitGroup, m *sync.Mutex) {
	m.Lock()
	globalblockChain = append(globalblockChain, block)
	m.Unlock()
	wg.Done()
}
func verifySig(nodes []Node, flagChan chan bool) {
	sigCount := 0
	for i := 0; i < len(nodes); i++ {
		if nodes[i].Sig {
			sigCount++
		}
	}
	if sigCount > int(flagArgs.totalF) {
		flagChan <- true
	}
}
func addNodeBlock(block FinalBlock, ID int, finalCommit chan bool, wg *sync.WaitGroup, m *sync.Mutex) {
	m.Lock()
	nodes[ID].blockChain = append(nodes[ID].blockChain, block)
	finalCommit <- true
	m.Unlock()
	wg.Done()
}
func PBFT(nodes []Node, commiteeMember []Node, runPBFT chan bool) {
	var w sync.WaitGroup
	var m sync.Mutex
	// 1.选leader
	rand.Seed(time.Now().UnixNano())
	leader := rand.Intn(int(flagArgs.n_per_commitee))
	// 2.leader出块
	w.Add(1)
	block := ProposedBlock{
		PreviousBlockHash: globalblockChain[len(globalblockChain)-1].Hash,
		Iteration:         uint(len(globalblockChain)),
		LeaderIP:          commiteeMember[leader].IP,
		Transactions:      getTxList(&w, &m),
	}
	// 坏节点有概率不进行操作
	// 3.验证
	// 把block广播给节点
	finalCommit := make(chan bool, 1000)
	for i := 0; i < int(flagArgs.n_per_commitee); i++ {
		go receivePre(commiteeMember, block, i, finalCommit)
	}
	// 收到f+1个提交信息
	for i := 0; i < int(flagArgs.totalF+1); i++ {
		<-finalCommit
	}
	// 4.随机生成最终委员会，节点验证签名数是否大于f+1
	var finalCommitee []Node
	for i := 0; i < int(flagArgs.c); i++ {
		finalID := rand.Intn(int(flagArgs.n)-1) + 1
		finalCommitee = append(finalCommitee, nodes[finalID])
	}
	// 验证签名数
	flagChan := make(chan bool, 1000)
	for i := 0; i < int(flagArgs.c); i++ {
		go verifySig(commiteeMember, flagChan)
	}
	for i := 0; i < int(flagArgs.c); i++ {
		<-flagChan
	}
	// 5.出块
	finalBlock := FinalBlock{
		Hash:          block.setHash(),
		ProposedBlock: block,
	}
	w.Add(1)
	addBlock(finalBlock, &w, &m)
	// 添加块
	for i := 1; i <= int(flagArgs.n); i++ {
		w.Add(1)
		go addNodeBlock(finalBlock, i, finalCommit, &w, &m)
	}
	for i := 1; i <= int(flagArgs.n); i++ {
		<-finalCommit
	}
	fmt.Printf("time.Now(): %v\n", time.Now())
	runPBFT <- true
}

func main() {
	// 1.获取参数信息

	flagArgs.CPUs = default_CPUs
	flagArgs.n = default_n
	flagArgs.totalF = default_totalF
	flagArgs.txNum = default_txNum
	flagArgs.txPerBlock = default_txPerBlock
	flagArgs.commitNum = default_commitNum
	flagArgs.n_per_commitee = default_n_per_commitee
	flagArgs.c = default_c
	runtime.GOMAXPROCS(int(flagArgs.CPUs))
	// 2.生成创世区块
	proposBlock := ProposedBlock{
		Iteration: 0,
		LeaderIP:  "0,0,0,0",
	}
	geneisBlock := FinalBlock{
		Hash:          proposBlock.setHash(),
		ProposedBlock: proposBlock,
	}
	globalblockChain = append(globalblockChain, geneisBlock)
	// 用户：1-n
	// 3.初始化用户
	// 随机坏蛋
	initNodes()
	// 4.生成交易集合
	txGen()
	for i := 1; i <= int(flagArgs.n); i++ {
		nodes[i].blockChain = append(nodes[i].blockChain, geneisBlock)
	}
	// 5.用户验证交易
	for len(txList) != 0 {
		Elastico()
	}
	// fmt.Printf("globalblockChain: %v\n", globalblockChain)
}
