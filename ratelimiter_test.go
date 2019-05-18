package ratelimiter

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	r := Rule{
		Name:         "rule1",
		IntervalInMs: 1000,
		CellNum:      2,
		LimitQps:     200,
	}
	Load(r)

	threadNum := 100

	go func() {
		for i:=0;i<threadNum;i++ {
			go func() {
				for {
					//Run(resource)
					succ,ctx := Entry()
					if succ {
						resource()
					}
					Exit(ctx)
				}
			}()
		}
	}()

	swit := make(chan struct{})
	cnt := 30
	tick := time.Tick(time.Second)
	go func() {
		for ;; {
			select {
			case <-tick:
				cnt--
				if cnt <= 0 {
					swit <- struct{}{}
				}
				fmt.Println("in second ", CurrTimeInMillis(), " p : ", rateLimiter.PassQps(), " s : ", rateLimiter.SuccessQps(), " b : ", rateLimiter.BlockQps(), " r : ", rateLimiter.AvgRt())
			}
		}
	}()

	<-swit

	fmt.Println("done", resCnt)
}

var resCnt = 0

func resource() {
	time.Sleep(time.Millisecond * time.Duration(rand.Intn(50)))
	resCnt++
}
