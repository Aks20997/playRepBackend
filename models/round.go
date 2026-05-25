package models

import (
	"time"

	"github.com/google/uuid"
)

// RoundState represents the state of a game round
type RoundState struct {
	Game       string    `json:"game"`
	RoundID    uuid.UUID `json:"roundId"`
	StartTS    int64     `json:"startTs"`    // unix milliseconds
	EndTS      int64     `json:"endTs"`      // unix milliseconds
	DurationMs int64     `json:"durationMs"` // duration in milliseconds
}

// GetTimeLeft calculates the time remaining in milliseconds
func (r *RoundState) GetTimeLeft() int64 {
	now := time.Now().UnixMilli()
	timeLeft := r.EndTS - now
	
	if timeLeft < 0 {
		return 0
	}
	
	return timeLeft
}

