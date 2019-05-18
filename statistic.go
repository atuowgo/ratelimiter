package ratelimiter

import (
	"sync/atomic"
)

type StatisticKind int

const (
	_ StatisticKind = iota - 1
	PASS
	BLOCK
	SUCCESS
	RT
)

var Kinds = []StatisticKind{
	PASS,
	BLOCK,
	SUCCESS,
	RT,
}

type atomicInt64 struct {
	cnt int64
}

func (ai *atomicInt64) reset() {
	atomic.StoreInt64(&ai.cnt, 0)
}

func (ai *atomicInt64) add(step int64) {
	atomic.AddInt64(&ai.cnt, step)
}

func (ai *atomicInt64) value() int64 {
	return atomic.LoadInt64(&ai.cnt)
}

type StatisticCell struct {
	collectors []*atomicInt64
}

func NewStatisticCell() *StatisticCell {
	collectors := make([]*atomicInt64, len(Kinds))
	for i := range Kinds {
		collectors[i] = new(atomicInt64)
	}
	return &StatisticCell{collectors: collectors}
}

func (cell *StatisticCell) Reset() *StatisticCell {
	for _, k := range Kinds {
		cell.collectors[k].reset()
	}
	return cell
}

func (cell *StatisticCell) ResetByCell(c *StatisticCell) *StatisticCell {
	for _, k := range Kinds {
		cell.collectors[k].reset()
		cell.collectors[k].add(c.Get(k))
	}
	return cell
}

func (cell *StatisticCell) Get(event StatisticKind) int64 {
	return cell.collectors[event].value()
}

func (cell *StatisticCell) Add(event StatisticKind, step int64) *StatisticCell {
	cell.collectors[event].add(step)
	return cell
}

func (cell *StatisticCell) Pass() int64 {
	return cell.collectors[PASS].value()
}

func (cell *StatisticCell) AddPass(n int64) {
	cell.Add(PASS, n)
}

func (cell *StatisticCell) Block() int64 {
	return cell.collectors[BLOCK].value()
}

func (cell *StatisticCell) AddBlock(n int64) {
	cell.Add(BLOCK, n)
}

func (cell *StatisticCell) Rt() int64 {
	return cell.collectors[RT].value()
}

func (cell *StatisticCell) AddRt(n int64) {
	cell.Add(RT, n)
}

func (cell *StatisticCell) Success() int64 {
	return cell.collectors[SUCCESS].value()
}

func (cell *StatisticCell) AddSuccess(n int64) {
	cell.Add(SUCCESS, n)
}

type Statistic interface {
	Success() int64
	MaxSuccess() int64
	Block() int64
	Pass() int64
	Rt() int64
	Details() []*StatisticDetail
	AddBlock(n int64)
	AddSuccess(n int64)
	AddPass(n int64)
	AddRt(n int64)
	IntervalInMs() int64
	CellStartTimeInMs() int64
}

type StatisticDetail struct {
	Timestamp  int64
	PassNum    int64
	BlockNum   int64
	SuccessNum int64
	Rt         int64
}

type StatisticSlidingWindow struct {
	SlidingWindow
}

func (ssw *StatisticSlidingWindow) NewEmptyCellValue(timeMillis int64) interface{} {
	return NewStatisticCell()
}

func (ssw *StatisticSlidingWindow) ResetCellTo(wrap *WindowCell, startTime int64) *WindowCell {
	wrap.ResetCell(startTime)
	wrap.Value.(*StatisticCell).Reset()
	return wrap
}

func NewDefSlidingWindow(cellNum, intervalInMs int64) *StatisticSlidingWindow {
	ssw := new(StatisticSlidingWindow)
	ssw.SlidingWindow = *NewSlidingWindow(cellNum, intervalInMs)
	ssw.SlidingWindowAction = ssw
	return ssw
}

type StatisticMetric struct {
	data *StatisticSlidingWindow
}

func NewStatisticMetric(cellNum, intervalInMs int64) *StatisticMetric {
	sw := NewDefSlidingWindow(cellNum, intervalInMs)
	return &StatisticMetric{
		data: sw,
	}
}

func (sm *StatisticMetric) Success() int64 {
	return sm.eventCnt(SUCCESS)
}

func (sm *StatisticMetric) MaxSuccess() int64 {
	sm.data.CurrentCell()
	var success int64
	for _, val := range sm.data.Values() {
		cell := val.(*StatisticCell)
		if success < cell.Success() {
			success = cell.Success()
		}
	}
	return success
}

func (sm *StatisticMetric) Block() int64 {
	return sm.eventCnt(BLOCK)
}

func (sm *StatisticMetric) Pass() int64 {
	return sm.eventCnt(PASS)
}

func (sm *StatisticMetric) Rt() int64 {
	return sm.eventCnt(RT)
}

func (sm *StatisticMetric) Details() []*StatisticDetail {
	details := make([]*StatisticDetail, 0)
	sm.data.CurrentCell()
	for _, window := range sm.data.List() {
		if window == nil {
			continue
		}
		val := window.Value.(*StatisticCell)
		detail := new(StatisticDetail)
		detail.BlockNum = val.Block()
		detail.PassNum = val.Pass()
		detail.SuccessNum = val.Success()
		detail.Rt = val.Rt()
		detail.Timestamp = window.CellStartTimeInMs

		details = append(details, detail)
	}

	return details
}

func (sm *StatisticMetric) AddBlock(n int64) {
	sm.addCnt(BLOCK, n)
}

func (sm *StatisticMetric) AddSuccess(n int64) {
	sm.addCnt(SUCCESS, n)
}

func (sm *StatisticMetric) AddPass(n int64) {
	sm.addCnt(PASS, n)
}

func (sm *StatisticMetric) AddRt(n int64) {
	sm.addCnt(RT, n)
}

func (sm *StatisticMetric) IntervalInMs() int64 {
	return sm.data.IntervalInMs
}

func (sm *StatisticMetric) CellStartTimeInMs() int64 {
	return sm.data.CellLengthInMs
}

func (sm *StatisticMetric) eventCnt(event StatisticKind) int64 {
	sm.data.CurrentCell()
	var cnt int64 = 0
	for _, val := range sm.data.Values() {
		cell := val.(*StatisticCell)
		cnt += cell.Get(event)
	}
	return cnt
}

func (sm *StatisticMetric) addCnt(event StatisticKind, n int64) {
	cell := sm.data.CurrentCell()
	cell.Value.(*StatisticCell).Add(event, n)
}
