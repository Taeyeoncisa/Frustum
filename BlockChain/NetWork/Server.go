package NetWork

import (
	"BlockChain/DB"
	"BlockChain/Message"
	"BlockChain/TxPool"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const SeedNodeURL = "172.16.2.214:10000"

// Latency 模拟延迟，单位毫秒
const Latency = 100

const CycleStatisticTime = 1

type Server struct {
	Logger   *zap.SugaredLogger
	ServerID string

	HttpServer *http.Server
	Peers      *NodeTable

	//网络服务退出通知管道
	ElegantExitCh chan struct{}
	//网络服务退出后再通知主进程退出
	NotifyMainProcess chan struct{}
	RecMsgCh          chan *Message.NodeMsg
	SenMsgCh          chan *Message.NodeMsg
	TxCache           TxPool.GenTxPool

	LoadStatus *Statistic
}

type Count struct {
	Flag    sync.Mutex
	Counter uint64
	Slice   []uint64
}

func NewCount() *Count {
	return &Count{
		Counter: 0,
		Slice:   make([]uint64, 0),
	}
}

type Statistic struct {
	Logger *zap.SugaredLogger

	StartTime time.Time
	EndTime   time.Time

	ExitCh chan struct{}

	CommitTx   *Count
	Traffic    *Count
	RunForOnce bool
	EndForOnce bool
}

func NewStatistic(logger *zap.SugaredLogger) *Statistic {
	return &Statistic{
		Logger:     logger,
		StartTime:  time.Time{},
		EndTime:    time.Time{},
		ExitCh:     make(chan struct{}, 1),
		CommitTx:   NewCount(),
		Traffic:    NewCount(),
		RunForOnce: false,
		EndForOnce: false,
	}
}

func NewServer(logger *zap.SugaredLogger,
	id string,
	port string,
	peers *NodeTable,
	rc, sc chan *Message.NodeMsg,
	notifyMain chan struct{}, txPool TxPool.GenTxPool) *Server {

	httpServer := new(http.Server)
	httpServer.Addr = GetNatIP() + ":" + port

	server := Server{
		Logger:            logger,
		ServerID:          id,
		HttpServer:        httpServer,
		Peers:             peers,
		ElegantExitCh:     make(chan struct{}, 1),
		NotifyMainProcess: notifyMain,
		RecMsgCh:          rc,
		SenMsgCh:          sc,
		LoadStatus:        NewStatistic(logger),
		TxCache:           txPool,
	}

	server.setRoute()
	server.Peers.AddNode("0-0", SeedNodeURL)

	return &server
}

func (s *Server) Start() {

	s.Logger.Infof("Server Start Listening AT %s", s.HttpServer.Addr)

	go s.Handle()

	if err := s.HttpServer.ListenAndServe(); err != nil {
		fmt.Println(err)
		return
	}

}

func (s *Server) Handle() {
	for {
		select {

		case msg := <-s.SenMsgCh:
			if ok, url := s.Peers.FindNode(msg.To); ok {
				go s.SendNodeMsg(url, msg)
			} else {
				s.Logger.Debugf("Could't Find Node %s", msg.To)
			}
		case <-s.ElegantExitCh:
			s.LoadStatus.EndTime = time.Now()
			close(s.LoadStatus.ExitCh)
			s.LoadStatus.GetResult()

			s.ShutDown()
			s.NotifyMainProcess <- struct{}{}
			return
		}
	}
}

func (s *Server) ShutDown() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := s.HttpServer.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed, err: %v\n", err)
	}

	s.Logger.Infof("server gracefully shutdown")

}

func (s *Server) setRoute() {
	http.HandleFunc("/NodeMsg", s.GetNodeMsg)
	http.HandleFunc("/Tx", s.GetTxs)
}

func (s *Server) GetNodeMsg(writer http.ResponseWriter, request *http.Request) {

	msg := new(Message.NodeMsg)
	err := json.NewDecoder(request.Body).Decode(msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	//读取完数据手动关闭可以提升效率节省资源，避免请求过多带来的问题
	defer request.Body.Close()

	s.RecMsgCh <- msg

	s.Logger.Debugf("%s -> %s %s, BlockHeight: %d",
		msg.From,
		msg.To,
		msg.MsgType,
		msg.BlockHeight,
	)

	s.LoadStatus.Handle(msg, uint64(request.ContentLength))
}

func (s *Server) GetTxs(writer http.ResponseWriter, request *http.Request) {

	tx := new(TxPool.Transaction)
	err := json.NewDecoder(request.Body).Decode(tx)
	if err != nil {
		fmt.Println(err)
		return
	}
	//读取完数据手动关闭可以提升效率节省资源，避免请求过多带来的问题
	defer request.Body.Close()

	s.TxCache.AddTx(tx)

}

func (s *Server) SendNodeMsg(url string, msg *Message.NodeMsg) {
	data := msg.Marshal()
	buff := bytes.NewBuffer(data)

	//代码内sleep模拟延迟
	time.Sleep((Latency + s.RandomLatency(20)) * time.Millisecond)

	res, err := http.Post("http://"+url+"/NodeMsg",
		"application/json", buff)
	//读取完数据手动关闭可以提升效率节省资源，避免请求过多带来的问题
	//若response的body为nil也可以不用调用此方法，习惯手动关闭
	defer res.Body.Close()
	if err != nil {
		log.Println(err)
		return
	}

	s.Logger.Infof("%s -> %s %s, BlockHeight: %d",
		msg.From,
		msg.To,
		msg.MsgType,
		msg.BlockHeight,
	)
	s.LoadStatus.Handle(msg, uint64(len(data)))
}

func (s *Server) RandomLatency(max int) time.Duration {
	rand.Seed(time.Now().Unix())
	return time.Duration(rand.Intn(max) + 1)

}

func (st *Statistic) Run() {

	if st.RunForOnce {
		//只启动一次
	} else {

		st.RunForOnce = true
		tick := time.NewTicker(CycleStatisticTime * time.Second)

		for {
			select {
			case <-tick.C:

				st.Traffic.Flag.Lock()
				st.Traffic.Slice = append(st.Traffic.Slice, st.Traffic.Counter)
				fmt.Println("Traffic:", st.Traffic.Counter, st.Traffic.Slice)
				st.Traffic.Counter = 0
				st.Traffic.Flag.Unlock()

			case <-st.ExitCh:
				//st.EndTime = time.Now()
				tick.Stop()

				return
			}
		}

	}

}

func (st *Statistic) Handle(msg *Message.NodeMsg, length uint64) {

	//流量统计从第一轮共识开始，初始化不统计
	if msg.BlockHeight >= 1 {
		st.Traffic.Flag.Lock()
		st.Traffic.Counter = st.Traffic.Counter + length
		st.Traffic.Flag.Unlock()

		//流量统计协程只在第一轮共识的pre-prepare阶段启动一次
		if msg.MsgType == "PrePrepare" && msg.BlockHeight == 1 && !st.RunForOnce {

			st.StartTime = time.Now()
			go st.Run()

		} else {
			//do nothing
		}

	} else {
		//do nothing
	}

}

func (st *Statistic) HandleBlock(block *DB.Block) {

	latestBlock := *block
	txNum := uint64(len(latestBlock.BlockBody))

	st.CommitTx.Flag.Lock()
	st.CommitTx.Counter = st.CommitTx.Counter + txNum

	var sum int64
	blockTime := time.Unix(0, latestBlock.TimeStamp)
	//fmt.Println(blockTime.Format("2006-01-02 15:04:05.000"))
	for _, tx := range latestBlock.BlockBody {
		txTime := time.Unix(0, tx.TimeStamp)
		//fmt.Println(txTime.Format("2006-01-02 15:04:05.000"))
		txLatency := blockTime.Sub(txTime).Milliseconds()
		//fmt.Println("TxLatency", txLatency)
		sum = sum + txLatency
	}

	st.CommitTx.Slice = append(st.CommitTx.Slice, uint64(sum)/txNum)

	st.CommitTx.Flag.Unlock()

	fmt.Println("CommitTx", st.CommitTx.Counter, st.CommitTx.Slice)
}

func (st *Statistic) GetResult() {

	times := st.EndTime.Sub(st.StartTime).Seconds()
	fmt.Printf("Duration Time: %f\n", times)

	if st.CommitTx.Counter != 0 {

		fmt.Printf("TPS: %f\n", float64(st.CommitTx.Counter)/times)

		var sum uint64
		for _, latency := range st.CommitTx.Slice {
			sum = sum + latency
		}
		fmt.Printf("Averange Latency: %d\n", sum/uint64(len(st.CommitTx.Slice)))

	}

	if len(st.Traffic.Slice) != 0 {
		var trafficSum uint64
		for _, traffic := range st.Traffic.Slice {
			trafficSum = trafficSum + traffic
		}
		fmt.Printf("NetWork Overload: %d\n", trafficSum/uint64(len(st.Traffic.Slice)))
	}

}

/*****************************************************************************************/

type NetWork interface {
	// SendMsg 发送消息方法，可能存在多种类型的addr，暂定为万能类型
	SendMsg(addr interface{}, msg *Message.NodeMsg) error
	// RecMsg 接收消息方法，将获得的消息传入接收管道
	RecMsg(rec chan *Message.NodeMsg, msg *Message.NodeMsg)
	// RecTx 接收交易方法，将获得的交易缓存至交易池
	RecTx(pool TxPool.GenTxPool, tx *TxPool.Transaction)

	Start()
	ShutDown()
}

type ServerV2 struct {
	Logger     *zap.SugaredLogger
	HttpServer *http.Server
	TxPool     TxPool.GenTxPool

	NotifyMainProcess chan struct{}
	RecMsgCh          chan *Message.NodeMsg
}

// NewServerV2 传入五个参数，日志打印器、监听地址（IP:PORT）、主进程维护的交易池、主进程获取子进程通知管道、主进程接收消息管道
func NewServerV2(log *zap.SugaredLogger,
	addr string,
	pool TxPool.GenTxPool,
	notify chan struct{},
	rec chan *Message.NodeMsg) *ServerV2 {

	httpserver := new(http.Server)
	httpserver.Addr = addr

	server := &ServerV2{
		Logger:            log,
		HttpServer:        httpserver,
		TxPool:            pool,
		NotifyMainProcess: notify,
		RecMsgCh:          rec,
	}

	return server
}

func (sv2 *ServerV2) SendMsg(addr interface{}, msg *Message.NodeMsg) error {
	data := msg.Marshal()
	buff := bytes.NewBuffer(data)

	//断言判断一下类型
	value, ok := addr.(string)
	if !ok {
		return errors.New("addr is not string")
	}

	res, err := http.Post("http://"+value+"/NodeMsg",
		"application/json", buff)
	//读取完数据手动关闭可以提升效率节省资源，避免请求过多带来的问题
	//若response的body为nil也可以不用调用此方法，习惯手动关闭
	defer res.Body.Close()
	if err != nil {
		log.Println(err)
		return err
	}

	sv2.Logger.Infof("%s -> %s %s, BlockHeight: %d",
		msg.From,
		msg.To,
		msg.MsgType,
		msg.BlockHeight,
	)
	return nil
}

func (sv2 *ServerV2) RecMsg(rec chan *Message.NodeMsg, msg *Message.NodeMsg) {

	select {
	case rec <- msg:
		sv2.Logger.Infof("[%s] -> [%s] %s, BlockHeight: %d",
			msg.From,
			msg.To,
			msg.MsgType,
			msg.BlockHeight)
	default:
		sv2.Logger.Debugf("RecMsgCh is Full")
	}

}

func (sv2 *ServerV2) RecTx(pool TxPool.GenTxPool, tx *TxPool.Transaction) {
	pool.AddTx(tx)
}

func (sv2 *ServerV2) setRoute() {
	http.HandleFunc("/NodeMsg", sv2.GetNodeMsg)
	http.HandleFunc("/Tx", sv2.GetTx)
}

func (sv2 *ServerV2) GetNodeMsg(writer http.ResponseWriter, request *http.Request) {
	msg := new(Message.NodeMsg)
	err := json.NewDecoder(request.Body).Decode(msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	//读取完数据手动关闭可以提升效率节省资源，避免请求过多带来的问题
	defer request.Body.Close()
	sv2.RecMsg(sv2.RecMsgCh, msg)
}

func (sv2 *ServerV2) GetTx(writer http.ResponseWriter, request *http.Request) {
	tx := new(TxPool.Transaction)
	err := json.NewDecoder(request.Body).Decode(tx)
	if err != nil {
		fmt.Println(err)
		return
	}
	//读取完数据手动关闭可以提升效率节省资源，避免请求过多带来的问题
	defer request.Body.Close()
	sv2.RecTx(sv2.TxPool, tx)
}

func (sv2 *ServerV2) Start() {
	sv2.Logger.Infof("Service Start Listening, Addr: %s", sv2.HttpServer.Addr)
	if err := sv2.HttpServer.ListenAndServe(); err != nil {
		log.Println(err)
		return
	}

}

func (sv2 *ServerV2) ShutDown() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := sv2.HttpServer.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed, err: %v\n", err)
	}

	sv2.Logger.Infof("server gracefully shutdown")
	//通知主进程
	sv2.NotifyMainProcess <- struct{}{}
}