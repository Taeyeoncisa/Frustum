type ProposedBlock struct {
	Hash              [32]byte       // 区块哈希值
	PreviousBlockHash [32]byte       // 前一个区块的哈希值
	Iteration         uint           // 区块编号
	TimeStamp         [32]byte       // 时间戳
	ShardID           [32]byte       // 区块所属分片
	LeaderPub         *PubKey        // 全局领导者公钥
	MerkleRoot        [32]byte       // 默克尔树根哈希
	Transactions      []*Transaction // 交易集合
}