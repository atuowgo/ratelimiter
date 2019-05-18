package ratelimiter

import "time"

func CurrTimeInMillis() int64 {
	return ToMillisFromNano(time.Now().UnixNano())
}

func ToMillisFromNano(timeNano int64) int64 {
	return timeNano / 1e6
}

func ToNamoFromMillis(timeMillis int64) int64 {
	return timeMillis * 1e6
}

func ToSecFromMillis(timeMillis int64) float64 {
	return float64(timeMillis) / 1000.0
}
