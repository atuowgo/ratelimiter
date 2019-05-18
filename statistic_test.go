package ratelimiter

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestAtomicInt64(t *testing.T) {
	var wg sync.WaitGroup
	val := new(atomicInt64)
	threadNum := 10000
	cntPerThread := 10000
	var targetNum int64 = 200
	for i := 0; i < threadNum; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < cntPerThread; j++ {
				if val.value() < targetNum {
					val.add(1)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	fmt.Println(val.value())
	if val.value() != targetNum {
		t.Error(fmt.Sprintf("expert %d but receive %d", threadNum*cntPerThread, val.value()))
	}
}

var wholeCntInSec atomicInt64

var globalPass atomicInt64
func TestStatisticSlidingWindow(t *testing.T) {
	slidingWindowInSecond := NewStatisticMetric(2, 1000)
	wholeCntInSec.reset()
	globalPass.reset()
	swit := make(chan struct{})
	go func() {
		for i := 0; i < 500; i++ {
			go func(idx int) {
				for ; ; {
					startTime := CurrTimeInMillis()
					if slidingWindowInSecond.Pass() + 1 > 20 {
						cost := CurrTimeInMillis() - startTime
						slidingWindowInSecond.AddBlock(1)
						slidingWindowInSecond.AddRt(cost)
					} else {
						slidingWindowInSecond.AddPass(1)
						resourceInSec()
						globalPass.add(1)
						slidingWindowInSecond.AddSuccess(1)
						cost := CurrTimeInMillis() - startTime
						slidingWindowInSecond.AddRt(cost)
					}
				}
			}(i)
		}
	}()

	cnt := 30
	tick := time.Tick(time.Second)
	go func() {
		var oldPass int64 = 0
		for {
			select {
			case <-tick:

				currPass := globalPass.value()
				oneSecPass := currPass - oldPass
				oldPass = currPass
				fmt.Println("in second ", CurrTimeInMillis(),slidingWindowInSecond.data.CurrentCell().CellStartTimeInMs,"op : ",oneSecPass, " p : ", slidingWindowInSecond.Pass(), " s : ", slidingWindowInSecond.Success(), " b : ", slidingWindowInSecond.Block(), " r : ", slidingWindowInSecond.Rt())
				cnt--
				if cnt <= 0 {
					swit <- struct{}{}
				}
			}
		}
	}()

	<-swit

	fmt.Println("done", wholeCntInSec)
	fmt.Println("global pass",globalPass)
}

func resourceInSec() {
	time.Sleep(time.Millisecond * time.Duration(rand.Intn(50)))
	wholeCntInSec.add(1)
}
