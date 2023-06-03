package Node

import (
	"crypto/rand"
	"math/big"
	"time"
)

// Tick 循环计时器
type Tick struct {
	Ticker      *time.Ticker
	CycleTime   time.Duration
	TimeOut     chan struct{}
	CycleDoneCh chan struct{}
}

func NewNodeTick() *Tick {

	returnValue := new(Tick)
	returnValue.TimeOut = make(chan struct{}, 1)
	returnValue.CycleDoneCh = make(chan struct{}, 1)

	return returnValue
}

// StartOrRestTick 启动或重置超时计时器
func (t *Tick) StartOrRestTick(cycleTime time.Duration) {
	result, _ := rand.Int(rand.Reader, big.NewInt(15))
	randomDuration := cycleTime + time.Duration(result.Int64()+1)*100

	if t.Ticker == nil {
		t.Ticker = time.NewTicker(randomDuration * time.Millisecond)
		t.CycleTime = randomDuration
	} else {
		t.Ticker.Reset(randomDuration * time.Millisecond)
		t.CycleTime = randomDuration
	}
}

// Run 循环检测启动
func (t *Tick) Run(cycleTime time.Duration) {
	t.StartOrRestTick(cycleTime)

	for {

		select {
		case <-t.Ticker.C:
			t.Ticker.Stop()
			t.TimeOut <- struct{}{}
			return

		case <-t.CycleDoneCh:
			return
		}

	}
}
