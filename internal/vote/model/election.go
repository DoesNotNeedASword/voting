package model

import (
	"encoding/json"
	"time"
)

type Election struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateElectionInput struct {
	Title       string
	Description string
	Type        string
	StartsAt    time.Time
	EndsAt      time.Time
	Status      string
}

type Ballot struct {
	ID          int64     `json:"id"`
	ElectionID  int64     `json:"election_id"`
	BallotType  string    `json:"ballot_type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateBallotInput struct {
	ElectionID  int64
	BallotType  string
	Title       string
	Description string
}

type BallotOption struct {
	ID              int64           `json:"id"`
	BallotID        int64           `json:"ballot_id"`
	SourceEntityID  int64           `json:"source_entity_id"`
	DisplayName     string          `json:"display_name"`
	Description     string          `json:"description"`
	ImageURL        string          `json:"image_url"`
	SnapshotPayload json.RawMessage `json:"snapshot_payload"`
	Position        int32           `json:"position"`
	CreatedAt       time.Time       `json:"created_at"`
}

type CreateBallotOptionInput struct {
	BallotID        int64
	SourceEntityID  int64
	DisplayName     string
	Description     string
	ImageURL        string
	SnapshotPayload json.RawMessage
	Position        int32
}

type CreatePartyBallotOptionInput struct {
	ElectionID        int64
	SourceEntityID    int64
	DisplayName       string
	Description       string
	ImageURL          string
	SnapshotPayload   json.RawMessage
	Position          int32
	BallotTitle       string
	BallotDescription string
}
