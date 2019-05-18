package ratelimiter

import (
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

type WindowCell struct {
	CellLengthInMs    int64
	CellStartTimeInMs int64
	Value             interface{}
}

func NewWindowCell(cellLengthInMs, cellStart int64, value interface{}) *WindowCell {
	return &WindowCell{
		CellLengthInMs:    cellLengthInMs,
		CellStartTimeInMs: cellStart,
		Value:             value,
	}
}

func (wrap *WindowCell) ResetCell(startTime int64) *WindowCell {
	wrap.CellStartTimeInMs = startTime
	return wrap
}

func (wrap *WindowCell) IsTimeInCell(timeMillis int64) bool {
	return wrap.CellStartTimeInMs <= timeMillis && timeMillis <= wrap.CellStartTimeInMs+wrap.CellLengthInMs
}

type SlidingWindowAction interface {
	NewEmptyCellValue(timeMillis int64) interface{}
	ResetCellTo(wrap *WindowCell, startTime int64) *WindowCell
}


const mutexLocked = 1 << iota
type Mutex struct {
	sync.Mutex
}
func (m *Mutex) TryLock() bool {
	return atomic.CompareAndSwapInt32((*int32)(unsafe.Pointer(&m.Mutex)), 0, mutexLocked)
}

type SlidingWindow struct {
	CellLengthInMs    int64
	CellNum           int64
	IntervalInMs      int64
	cells             []*WindowCell
	genLock           Mutex
	updateLock        Mutex
	SlidingWindowAction
}

func NewSlidingWindow(cellNum, intervalInMs int64) *SlidingWindow {
	IsTrue(cellNum > 0, fmt.Sprintf("cell count is invalid: %d", cellNum))
	IsTrue(intervalInMs > 0, "total time interval of the sliding window should be positive")
	IsTrue(intervalInMs%cellNum == 0, "time span needs to be evenly divided")

	return &SlidingWindow{
		CellLengthInMs: intervalInMs / cellNum,
		IntervalInMs:   intervalInMs,
		CellNum:        cellNum,
		cells:          make([]*WindowCell, cellNum),
	}
}

func (sw *SlidingWindow) CurrentCell() *WindowCell {
	cell := sw.CurrentCellByTime(CurrTimeInMillis())
	return cell
}

func (sw *SlidingWindow) CurrentCellByTime(timeMillis int64) *WindowCell {
	if timeMillis < 0 {
		return nil
	}

	var (
		idx           = sw.cellIdxByTime(timeMillis)
		cellStartTime = sw.cellStartTimeByTime(timeMillis)
	)

	for ; ; {
		old := sw.cells[idx]
		if old == nil {
			if sw.genLock.TryLock() {
				cell := NewWindowCell(sw.CellLengthInMs, cellStartTime, sw.NewEmptyCellValue(timeMillis))
				sw.cells[idx] = cell
				sw.genLock.Unlock()
			}else{
				continue
			}
			return sw.cells[idx]
		} else if cellStartTime == old.CellStartTimeInMs {
			return old
		} else if cellStartTime > old.CellStartTimeInMs {
			var ret *WindowCell
			if sw.updateLock.TryLock() {
				 ret = sw.ResetCellTo(old, cellStartTime)
				sw.updateLock.Unlock()
			}else{
				continue
			}
			return ret
		} else if cellStartTime < old.CellStartTimeInMs {
			return NewWindowCell(sw.CellLengthInMs, cellStartTime, sw.NewEmptyCellValue(timeMillis))
		}
	}
}

//calculate given time belong to which cell
func (sw *SlidingWindow) cellIdxByTime(timeMillis int64) int {
	timeId := timeMillis / sw.CellLengthInMs
	return int(timeId % int64(len(sw.cells)))
}

//calculate start time of the cell from given time
func (sw *SlidingWindow) cellStartTimeByTime(timeMillis int64) int64 {
	return timeMillis - timeMillis%sw.CellLengthInMs
}

func (sw *SlidingWindow) GetPreviousCellByTime(timeMillis int64) *WindowCell {
	if timeMillis < 0 {
		return nil
	}

	timeId := (timeMillis - sw.CellLengthInMs) / sw.CellLengthInMs
	idx := int(timeId % int64(len(sw.cells)))
	timeMillis = timeMillis - sw.CellLengthInMs
	cell := sw.cells[idx]

	if cell == nil || sw.IsCellDeprecated(cell) {
		return nil
	}

	if cell.CellStartTimeInMs+sw.CellLengthInMs < timeMillis {
		return nil
	}
	return cell
}

func (sw *SlidingWindow) GetPreviousCell() *WindowCell {
	return sw.GetPreviousCellByTime(CurrTimeInMillis())
}

func (sw *SlidingWindow) GetCellValue(timeMillis int64) interface{} {
	if timeMillis < 0 {
		return nil
	}

	idx := sw.cellIdxByTime(timeMillis)
	cell := sw.cells[idx]

	if cell == nil || !cell.IsTimeInCell(timeMillis) {
		return nil
	}

	return cell.Value
}

//is the cell not in window
func (sw *SlidingWindow) IsCellDeprecated(cell *WindowCell) bool {
	return sw.IsCellDeprecatedByTime(CurrTimeInMillis(), cell)
}

func (sw *SlidingWindow) IsCellDeprecatedByTime(timeMillis int64, cell *WindowCell) bool {
	if timeMillis > cell.CellStartTimeInMs {
		return timeMillis-cell.CellStartTimeInMs > sw.IntervalInMs
	}else{
		return cell.CellStartTimeInMs - timeMillis > sw.IntervalInMs
	}
	//return timeMillis >= cell.CellStartTimeInMs
}

func (sw *SlidingWindow) List() []*WindowCell {
	return sw.ListByTime(CurrTimeInMillis())
}

func (sw *SlidingWindow) ListByTime(timeMillis int64) []*WindowCell {
	size := len(sw.cells)
	cells := make([]*WindowCell, 0)
	for i := 0; i < size; i++ {
		cell := sw.cells[i]
		if cell == nil || sw.IsCellDeprecatedByTime(timeMillis, cell) {
			continue
		}
		cells = append(cells, cell)
	}
	return cells
}

func (sw *SlidingWindow) ListAll() []*WindowCell {
	size := len(sw.cells)
	cells := make([]*WindowCell, 0)
	for i := 0; i < size; i++ {
		cell := sw.cells[i]
		if cell == nil {
			continue
		}
		cells = append(cells, cell)
	}
	return cells
}

func (sw *SlidingWindow) Values() []interface{} {
	return sw.ValuesByTime(CurrTimeInMillis())
}

func (sw *SlidingWindow) ValuesByTime(timeMillis int64) []interface{} {
	if timeMillis < 0 {
		return make([]interface{}, 0)
	}
	size := len(sw.cells)
	values := make([]interface{}, 0)
	for i := 0; i < size; i++ {
		cell := sw.cells[i]
		if cell == nil || sw.IsCellDeprecatedByTime(timeMillis, cell) {
			continue
		}
		values = append(values, cell.Value)
	}
	return values
}

func (sw *SlidingWindow) GetIntervalInSecond() float64 {
	return ToSecFromMillis(sw.IntervalInMs)
}
