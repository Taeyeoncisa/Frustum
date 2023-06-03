package main

// 设置默认参数

// 使用CPU个数
const default_CPUs uint = 16

// 节点个数
const default_n uint = 128

// 分片个数
const default_commitNum uint = 16

// 每个分片委员会个数
const default_n_per_commitee uint = default_n / default_commitNum

// 坏节点比例
const default_totalF uint = default_n / default_commitNum / 4

// 交易个数
const default_txNum uint = 1024

// 每个区块包含交易数目
const default_txPerBlock uint = 8

// 最终委员会数量
const default_c uint = 10

// 参数信息
type FlagArgs struct {
	CPUs           uint
	n              uint
	totalF         uint
	txNum          uint
	txPerBlock     uint
	commitNum      uint
	n_per_commitee uint
	c              uint
}
