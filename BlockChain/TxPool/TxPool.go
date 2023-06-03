package TxPool

import (
	"log"
	"sync"
	"time"
)

const TcPoolChLength = 1 << 20

// IdealSendingTxTPS 交易池中交易到达率（理想），通过定时循环向管道发送交易模拟，存在较明显误差
const IdealSendingTxTPS = 3000

type GenTxPool interface {
	// AddTx 将交易加入交易池
	AddTx(tx *Transaction)
	// CommitTrs 打包一定数量的交易
	CommitTrs(txNumInBlock int) *[]Transaction
	// CycleAddTx 模拟生成交易，stopWorker管道通知协程停止，shardNum为分片总数
	CycleAddTx(stopWorker chan struct{}, shardNum int)
}

// TxPool 交易池结构体，目前暂定三个等级的缓存管道
type TxPool struct {
	Channels map[int]chan *Transaction
	Priority bool
}

// NewTxPool 实例化TxPool，传入两个参数是否区别优先级，有几个优先级
func NewTxPool(priority bool, priorityNum int) *TxPool {

	txPool := new(TxPool)
	txPool.Channels = make(map[int]chan *Transaction)
	txPool.Priority = priority

	if priority {
		for i := 0; i < priorityNum; i++ {
			txPool.Channels[i] = make(chan *Transaction, TcPoolChLength)
		}
	} else {
		txPool.Channels[0] = make(chan *Transaction, TcPoolChLength*priorityNum)
	}

	return txPool

}

// PoolLength TxPool各个等级交易缓存长度和总长度
type PoolLength struct {
	LevelLength []int
	TotalLength int
}

// ChLength 计算某一时刻时，TxPool中缓存状态
func (tp *TxPool) ChLength() *PoolLength {

	result := new(PoolLength)
	result.LevelLength = make([]int, len(tp.Channels))
	for k, v := range tp.Channels {
		result.LevelLength[k] = len(v)
		result.TotalLength = result.TotalLength + result.LevelLength[k]
	}

	return result
}

// AddTx AddTr 根据不同的共识协议类型来将交易写入交易池
func (tp *TxPool) AddTx(tx *Transaction) {

	if tp.Priority {
		select {
		case tp.Channels[int(tx.Level)] <- tx:
		default:
			// channel is full
		}
	} else {
		select {
		case tp.Channels[0] <- tx:
		default:
			// channel is full
		}
	}

}

// CommitTrs 根据不同的共识协议类型，提供不同打包交易方法
func (tp *TxPool) CommitTrs(txNumInBlock int) *[]Transaction {

	newBatch := new([]Transaction)
	currentPoolLen := tp.ChLength()

	if currentPoolLen.TotalLength == 0 {
		return tp.CommitTrs(txNumInBlock)
	}

	num := tp.LengthCompare(currentPoolLen, txNumInBlock)

	switch num {
	//0等级的交易数量大于TrNumInBlock
	case 0:
		for i := 0; i < txNumInBlock; i++ {
			tr := <-tp.Channels[0]
			*newBatch = append(*newBatch, *tr)
		}
	//交易池总量小于TrNumInBlock
	case len(currentPoolLen.LevelLength):
		//遍历所有管道
		for k, v := range currentPoolLen.LevelLength {
			//只抽取采集长度时刻的取值数量的交易，因为交易是不断抵达，长度在时刻变化
			for i := 0; i < v; i++ {
				tr := <-tp.Channels[k]
				*newBatch = append(*newBatch, *tr)
			}
		}
	//介于上述两者之间
	default:
		left := txNumInBlock
		for i := 0; i < num+1; i++ {
			//遍历到最后一个管道时，提取的是剩余要提取的数量
			if i == num {
				for j := 0; j < left; j++ {
					tr := <-tp.Channels[i]
					*newBatch = append(*newBatch, *tr)
				}
			} else {
				//只抽取采集长度时刻的取值数量的交易，因为交易是不断抵达，长度在时刻变化
				for p := 0; p < currentPoolLen.LevelLength[i]; p++ {
					tr := <-tp.Channels[i]
					*newBatch = append(*newBatch, *tr)
				}
				//每提取一个管道，计算一次剩余数量
				left = left - currentPoolLen.LevelLength[i]
			}
		}
	}

	return newBatch
}

// LengthCompare 判断交易池缓存数量和区块交易数量的大小，
// 交易池中从0~返回值等级的交易数量总和大于等于TrNumInBlock，
// 若返回值大于最高等级，则表示交易池总量低于TrNumInBlock
func (tp *TxPool) LengthCompare(l *PoolLength, trNumInBlock int) int {
	count := 0
	for k, v := range l.LevelLength {
		count = count + v
		if count >= trNumInBlock {
			return k
		}
	}
	return len(l.LevelLength)
}

func (tp *TxPool) CycleAddTx(stop chan struct{}, shardNum int) {
	workerNum := IdealSendingTxTPS / 1000
	for i := 0; i < workerNum; i++ {
		go tp.SendTxWorker(i, stop, shardNum)
	}
}

func (tp *TxPool) SendTxWorker(workerID int, stop chan struct{}, shardNum int) {

	sendTxTick := time.NewTicker(1 * time.Millisecond)
	tpsTick := time.NewTicker(10 * time.Second)
	count := 0
	//count重置清零和累加操作可能存在并发问题
	var flag sync.Mutex

	for {
		select {

		case <-sendTxTick.C:
			tp.AddTx(NewTransaction(shardNum))
			flag.Lock()
			count++
			flag.Unlock()
		case <-tpsTick.C:
			log.Printf("Worker-%d's Actual TPS is: %d", workerID, count/10)
			flag.Lock()
			count = 0
			flag.Unlock()
		case <-stop:
			return
		}
	}
}
