package DB

import (
	"BlockChain/TxPool"
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"os"
	"strconv"
	"time"
)

const dbFile = "Blockchain_%s.db"
const blocksBucket = "Blocks"

// Block 区块结构体
type Block struct {
	TimeStamp   int64
	BlockHeight uint64
	Miner       string //挖掘区块的矿工ID

	PreBlockHash []byte
	BlockHash    []byte

	BlockBody []TxPool.Transaction
}

// Serialize 区块序列化为可存储的[]byte
func (bk *Block) Serialize() []byte {

	result := new(bytes.Buffer)
	encoder := gob.NewEncoder(result)

	err := encoder.Encode(bk)
	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}

// DeserializeBlock []byte数据反序列化为Block结构体
func DeserializeBlock(data []byte) *Block {

	block := new(Block)

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(block)
	if err != nil {
		log.Panic(err)
	}
	return block
}

// SetHash 新建区块时计算区块哈希
func (bk *Block) SetHash() {

	data1 := strconv.FormatInt(bk.TimeStamp, 10)
	data2 := strconv.FormatUint(bk.BlockHeight, 10)

	var data3 [][]byte
	for _, tr := range bk.BlockBody {

		data3 = append(data3, tr.Serialize())

	}

	hash := sha256.Sum256(bytes.Join([][]byte{
		[]byte(data1),
		[]byte(data2),
		[]byte(bk.Miner),
		bk.PreBlockHash,
		bytes.Join(data3, []byte{}),
	}, []byte{}))

	bk.BlockHash = hash[:]
}

// CalculateTrLatency 计算区块体中交易的提交延迟，输入一个统计不同优先级交易的map数据
//func (bk *Block) CalculateTrLatency(trLatency map[uint]*[]float64) {
//
//	for _, v := range bk.BlockBody {
//
//		latency := TimeStampCompare(v.TimeStamp, bk.TimeStamp).Milliseconds()
//
//		if value, ok := trLatency[v.Level]; ok {
//
//			*value = append(*value, float64(latency))
//
//		} else {
//
//			newArray := new([]float64)
//			*newArray = append(*newArray, float64(latency))
//			trLatency[v.Level] = newArray
//
//		}
//
//	}
//
//}

// BlockChain 内存中维护的区块链结构体
type BlockChain struct {
	Tip      []byte //最后一个区块的哈希
	DataBase *bolt.DB

	LatestBlockIndex uint64
}

type BlockChainIterator struct {
	currentHash []byte
	db          *bolt.DB
}

func (bci *BlockChainIterator) Next() *Block {

	block := new(Block)

	err := bci.db.View(func(tx *bolt.Tx) error {

		bucket := tx.Bucket([]byte(blocksBucket))
		blockData := bucket.Get(bci.currentHash)
		if blockData == nil {
			fmt.Printf("block don't exit. Hash: %x", bci.currentHash)
		} else {
			block = DeserializeBlock(blockData)
		}

		return nil
	})
	check(err)
	bci.currentHash = block.PreBlockHash
	return block
}

func check(err error) {

	if err != nil {
		log.Panic(err)
	}

}

func dbExits(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

// NewBlockChain 根据nodeID创建新区块链数据
func NewBlockChain(nodeID string) *BlockChain {

	blockchain := new(BlockChain)

	dbFileName := fmt.Sprintf(dbFile, nodeID)
	if dbExits(dbFileName) {

		fmt.Printf("%s already exists.\n", dbFileName)
		db, err := bolt.Open(dbFileName, 0600, nil)
		check(err)

		err = db.Update(func(tx *bolt.Tx) error {

			bucket, err2 := tx.CreateBucketIfNotExists([]byte(blocksBucket))
			check(err2)

			latestBlockHash := bucket.Get([]byte("latest"))
			//区块链为空，无数据；反之提取最新区块信息
			if latestBlockHash == nil {
				fmt.Println("No data in blockchain")
				return nil
			} else {

				latestBlock := bucket.Get(latestBlockHash)
				if string(latestBlock) == "preCommitBlockHash" {
					blockchain.Tip = latestBlockHash
					fmt.Println("latestBlock is PreCommitBlock, don't Know Index")
				} else {
					blockchain.Tip = latestBlockHash
					block := DeserializeBlock(latestBlock)
					blockchain.LatestBlockIndex = block.BlockHeight
				}

			}

			return nil
		})
		check(err)
		blockchain.DataBase = db
		return blockchain

	} else {

		//fmt.Printf("No BlockChain exist, creat %s\n", dbFileName)
		db, err := bolt.Open(dbFileName, 0600, nil)
		check(err)

		err = db.Update(func(tx *bolt.Tx) error {

			_, err2 := tx.CreateBucketIfNotExists([]byte(blocksBucket))
			check(err2)

			return nil
		})
		check(err)
		blockchain.DataBase = db
		return blockchain
	}

}

func (bc *BlockChain) Iterator() *BlockChainIterator {

	bci := new(BlockChainIterator)
	bci.currentHash = bc.Tip
	bci.db = bc.DataBase

	return bci
}

func (bc *BlockChain) AddNewBlock(block *Block) {

	err := bc.DataBase.Update(func(tx *bolt.Tx) error {

		bucket := tx.Bucket([]byte(blocksBucket))
		checkIfExist := bucket.Get(block.BlockHash)
		//确认区块是否已存在
		if checkIfExist == nil {
			//区块写入数据库
			err := bucket.Put(block.BlockHash, block.Serialize())
			check(err)
			//更新数据库中最新区块哈希和内存中BlockChain状态
			bc.Tip = block.BlockHash
			err = bucket.Put([]byte("latest"), block.BlockHash)
			check(err)

		} else {
			//区块存在，但为预提交区块，只保存了哈希
			if string(checkIfExist) == "preCommitBlockHash" {
				fmt.Printf("PreCommitBlock become committed, blockhash is: %s\n", block.BlockHash)
				err := bucket.Put(block.BlockHash, block.Serialize())
				check(err)

			} else {
				//区块存在且为完整区块
				fmt.Printf("Block has exist, blockhash is: %s\n", block.BlockHash)

			}

		}

		return nil
	})
	check(err)
}

// AddNewBlockHash 预提交区块，键为区块哈希，值为“preCommitBlockHash”
func (bc *BlockChain) AddNewBlockHash(hash []byte) {

	err := bc.DataBase.Update(func(tx *bolt.Tx) error {

		bucket := tx.Bucket([]byte(blocksBucket))
		checkIfExist := bucket.Get(hash)
		if checkIfExist == nil {

			err := bucket.Put(hash, []byte("preCommitBlockHash"))
			check(err)

			bc.Tip = hash
			err = bucket.Put([]byte("latest"), hash)
			check(err)

		} else {

			fmt.Printf("Block has exist, blockhash is: %s\n", hash)

		}

		return nil
	})
	check(err)

}

// CalculateTPSandLatency 与Iterator方法结合使用，计算区块链中所有交易的提交延迟和系统性能
//func (bc *BlockChain) CalculateTPSandLatency() bool {
//
//	bci := bc.Iterator()
//	blockCount := 0
//	var latestBlock *Block
//	var firstBlock *Block
//
//	trLatency := make(map[uint]*[]float64)
//	var totalTr []float64
//
//	fmt.Printf("Start Analysis BlockChain\n")
//
//	for {
//
//		block := bci.Next()
//		if blockCount == 0 {
//			latestBlock = block
//			fmt.Printf("LatestBlock, BlockHeight:%d, BlockHash: %x\n",
//				latestBlock.BlockHeight,
//				latestBlock.BlockHash)
//		}
//
//		block.CalculateTrLatency(trLatency)
//
//		blockCount++
//		//抵达创世区块，结束循环
//		if len(block.PreBlockHash) == 0 {
//			firstBlock = block
//			fmt.Printf("FirstBlock, BlockHeight:%d, BlockHash: %x\n",
//				firstBlock.BlockHeight,
//				firstBlock.BlockHash)
//			break
//		}
//	}
//
//	//确定遍历的个数和区块高度差一致
//	if uint64(blockCount) == (latestBlock.BlockHeight - firstBlock.BlockHeight + 1) {
//		fmt.Printf("BlockChain has %d Blocks\n", blockCount)
//	} else {
//		fmt.Printf("Count don't match BlockHeight difference, use latter\n")
//		fmt.Printf("BlockChain has %d Blocks\n", blockCount)
//	}
//
//	for k, v := range trLatency {
//
//		trAverageLatency := maths.ArithMean(&maths.Vector{
//			A: *v,
//			L: len(*v),
//		})
//		fmt.Printf("Level %d Tr's Average Latency is: %f, num:%d\n",
//			k, trAverageLatency, len(*v))
//		totalTr = append(totalTr, *v...)
//	}
//	totalTrAverageLatency := maths.ArithMean(&maths.Vector{
//		A: totalTr,
//		L: len(totalTr),
//	})
//	totalTime := TimeStampCompare(firstBlock.TimeStamp, latestBlock.TimeStamp).Seconds()
//	fmt.Printf("Tr's Average Latency: %f, System TPS: %f\n",
//		totalTrAverageLatency,
//		float64(len(totalTr))/(totalTime))
//
//	return true
//}

// TimeStampCompare 时间戳计算提交延迟
func TimeStampCompare(smallTimeStamp, bigTimeStamp int64) time.Duration {

	time1 := time.Unix(0, smallTimeStamp)
	time2 := time.Unix(0, bigTimeStamp)

	duration := time2.Sub(time1)

	return duration

}
