package utils

import "time"

// CalculateTimeLeft calculates the time remaining in milliseconds from start and end timestamps
func CalculateTimeLeft(startTS, endTS int64) int64 {
	now := time.Now().UnixMilli()
	timeLeft := endTS - now
	
	if timeLeft < 0 {
		return 0
	}
	
	return timeLeft
}

// GetTimeLeftSeconds returns time left in seconds
func GetTimeLeftSeconds(startTS, endTS int64) int64 {
	return CalculateTimeLeft(startTS, endTS) / 1000
}

