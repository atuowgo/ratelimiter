package ratelimiter

import (
	"context"
)

type Qps interface {
	TotalRequest() int64
	TotalPass() int64
	TotalSuccess() int64
	BlockRequest() int64
	PassQps() float64
	BlockQps() float64
	TotalQps() float64
	SuccessQps() float64
	MaxSuccessQps() float64
	AvgRt() float64
	AddPass(count int64)
	AddRtAndSuccess(rt, success int64)
	AddBlock(count int64)
	Reset()
}

type RateLimiter struct {
	window *StatisticMetric
	Rule   Rule
}

func NewRateLimiter(rule Rule) *RateLimiter {
	return &RateLimiter{
		window: NewStatisticMetric(rule.CellNum, rule.IntervalInMs),
		Rule:   rule,
	}
}

func (limiter *RateLimiter) TotalRequest() int64 {
	totalRequest := limiter.window.Pass() + limiter.window.Block()
	return totalRequest
}

func (limiter *RateLimiter) TotalPass() int64 {
	return limiter.window.Pass()
}

func (limiter *RateLimiter) TotalSuccess() int64 {
	return limiter.window.Success()
}

func (limiter *RateLimiter) BlockRequest() int64 {
	return limiter.window.Block()
}

func (limiter *RateLimiter) PassQps() float64 {
	return float64(limiter.window.Pass()) / limiter.windowIntervalInSec()
}

func (limiter *RateLimiter) BlockQps() float64 {
	return float64(limiter.window.Block()) / limiter.windowIntervalInSec()
}

func (limiter *RateLimiter) TotalQps() float64 {
	return limiter.PassQps() + limiter.BlockQps()
}

func (limiter *RateLimiter) SuccessQps() float64 {
	return float64(limiter.window.Success()) / limiter.windowIntervalInSec()
}

func (limiter *RateLimiter) MaxSuccessQps() float64 {
	return float64(limiter.window.MaxSuccess()) * float64(limiter.window.data.CellNum)
}

func (limiter *RateLimiter) windowIntervalInSec() float64 {
	return limiter.window.data.GetIntervalInSecond()
}

func (limiter *RateLimiter) AvgRt() float64 {
	var succNum = limiter.window.Success()
	if succNum == 0 {
		return 0.0
	}
	return float64(limiter.window.Rt()) / float64(succNum)
}

func (limiter *RateLimiter) AddPass(count int64) {
	limiter.window.AddPass(count)
}

func (limiter *RateLimiter) AddRtAndSuccess(rt, success int64) {
	limiter.window.AddSuccess(success)
	limiter.window.AddRt(rt)
}

func (limiter *RateLimiter) AddBlock(count int64) {
	limiter.window.AddBlock(count)
}

func (limiter *RateLimiter) Reset() {
	limiter = NewRateLimiter(limiter.Rule)
}

type Rule struct {
	Name         string
	IntervalInMs int64
	CellNum      int64
	LimitQps     int64
}

func Load(r Rule) {
	rateLimiter = NewRateLimiter(r)
}

var ctx context.Context
var rateLimiter *RateLimiter

type rateLimiterContext struct {
	startTime   int64
	limitStatus bool
}

const key_limiter_ctx = "_limiter_ctx_"

func Entry() (bool, context.Context) {
	var status = true
	startTime := CurrTimeInMillis()
	if checkPass() {
		rateLimiter.AddPass(1)
	} else {
		rateLimiter.AddBlock(1)
		status = false
	}
	subCtx := context.WithValue(ctx, key_limiter_ctx, rateLimiterContext{startTime: startTime, limitStatus: status})
	return status, subCtx
}

func Exit(subCtx context.Context) {
	limitCtx := subCtx.Value(key_limiter_ctx).(rateLimiterContext)
	if limitCtx.limitStatus {
		rt := CurrTimeInMillis() - limitCtx.startTime
		rateLimiter.AddRtAndSuccess(rt, 1)
	}
}

func checkPass() bool {
	return int64(rateLimiter.PassQps()) < rateLimiter.Rule.LimitQps
}

func Run(f func()) {
	startTime := CurrTimeInMillis()
	if checkPass() {
		rateLimiter.AddPass(1)
		f()
		rateLimiter.AddRtAndSuccess(CurrTimeInMillis()-startTime, 1)
	} else {
		rateLimiter.AddBlock(1)
	}
}
