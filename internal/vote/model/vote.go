package model

import "time"

type Vote struct {
	ID             int64     `json:"id"`
	BallotID       int64     `json:"ballot_id"`
	BallotOptionID int64     `json:"ballot_option_id"`
	UserID         int64     `json:"user_id"`
	CreatedAt      time.Time `json:"created_at"`
}
