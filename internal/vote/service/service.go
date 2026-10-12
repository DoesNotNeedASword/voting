package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"Voting/internal/vote/model"
	"Voting/internal/vote/repository"
)

var ErrInvalidInput = errors.New("invalid vote input")

type Repository interface {
	CreateElection(ctx context.Context, input model.CreateElectionInput) (model.Election, error)
	CreateBallot(ctx context.Context, input model.CreateBallotInput) (model.Ballot, error)
	CreateBallotOption(ctx context.Context, input model.CreateBallotOptionInput) (model.BallotOption, error)
	CreatePartyBallotOption(ctx context.Context, input model.CreatePartyBallotOptionInput) (model.BallotOption, error)
	CreatePartyVote(ctx context.Context, userID int64, partyID int64) (model.Vote, error)
}

type VoteService struct {
	repository Repository
}

func New(repository Repository) *VoteService {
	return &VoteService{repository: repository}
}

func (s *VoteService) CreateElection(ctx context.Context, input model.CreateElectionInput) (model.Election, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Type = strings.TrimSpace(input.Type)
	input.Status = strings.TrimSpace(input.Status)
	if input.Status == "" {
		input.Status = "draft"
	}

	if input.Title == "" ||
		!isElectionTypeValid(input.Type) ||
		!isElectionStatusValid(input.Status) ||
		input.StartsAt.IsZero() ||
		input.EndsAt.IsZero() ||
		!input.StartsAt.Before(input.EndsAt) {
		return model.Election{}, ErrInvalidInput
	}

	return s.repository.CreateElection(ctx, input)
}

func (s *VoteService) CreateBallot(ctx context.Context, input model.CreateBallotInput) (model.Ballot, error) {
	input.BallotType = strings.TrimSpace(input.BallotType)
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.ElectionID <= 0 || !isBallotTypeValid(input.BallotType) || input.Title == "" {
		return model.Ballot{}, ErrInvalidInput
	}

	return s.repository.CreateBallot(ctx, input)
}

func (s *VoteService) CreateBallotOption(ctx context.Context, input model.CreateBallotOptionInput) (model.BallotOption, error) {
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Description = strings.TrimSpace(input.Description)
	input.ImageURL = strings.TrimSpace(input.ImageURL)
	if len(input.SnapshotPayload) == 0 {
		input.SnapshotPayload = json.RawMessage(`{}`)
	}

	if input.BallotID <= 0 || input.SourceEntityID <= 0 || input.DisplayName == "" || !json.Valid(input.SnapshotPayload) {
		return model.BallotOption{}, ErrInvalidInput
	}

	return s.repository.CreateBallotOption(ctx, input)
}

func (s *VoteService) CreatePartyBallotOption(ctx context.Context, input model.CreatePartyBallotOptionInput) (model.BallotOption, error) {
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Description = strings.TrimSpace(input.Description)
	input.ImageURL = strings.TrimSpace(input.ImageURL)
	input.BallotTitle = strings.TrimSpace(input.BallotTitle)
	input.BallotDescription = strings.TrimSpace(input.BallotDescription)
	if input.BallotTitle == "" {
		input.BallotTitle = "Party ballot"
	}
	if input.BallotDescription == "" {
		input.BallotDescription = "Vote for party"
	}
	if len(input.SnapshotPayload) == 0 {
		input.SnapshotPayload = json.RawMessage(`{}`)
	}

	if input.ElectionID <= 0 ||
		input.SourceEntityID <= 0 ||
		input.DisplayName == "" ||
		!json.Valid(input.SnapshotPayload) {
		return model.BallotOption{}, ErrInvalidInput
	}

	return s.repository.CreatePartyBallotOption(ctx, input)
}

func (s *VoteService) VoteForParty(ctx context.Context, userID int64, partyID int64) (model.Vote, error) {
	if userID <= 0 || partyID <= 0 {
		return model.Vote{}, ErrInvalidInput
	}

	vote, err := s.repository.CreatePartyVote(ctx, userID, partyID)
	if err != nil {
		return model.Vote{}, err
	}

	return vote, nil
}

func IsAlreadyVoted(err error) bool {
	return errors.Is(err, repository.ErrAlreadyVoted)
}

func IsNoActivePartyBallot(err error) bool {
	return errors.Is(err, repository.ErrNoActivePartyBallot)
}

func IsPartyOptionNotFound(err error) bool {
	return errors.Is(err, repository.ErrPartyOptionNotFound)
}

func IsDuplicateBallot(err error) bool {
	return errors.Is(err, repository.ErrDuplicateBallot)
}

func IsDuplicateOption(err error) bool {
	return errors.Is(err, repository.ErrDuplicateOption)
}

func IsElectionNotFound(err error) bool {
	return errors.Is(err, repository.ErrElectionNotFound)
}

func IsBallotNotFound(err error) bool {
	return errors.Is(err, repository.ErrBallotNotFound)
}

func isElectionTypeValid(value string) bool {
	switch value {
	case "parliamentary", "presidential", "mixed":
		return true
	default:
		return false
	}
}

func isElectionStatusValid(value string) bool {
	switch value {
	case "draft", "active", "finished", "cancelled":
		return true
	default:
		return false
	}
}

func isBallotTypeValid(value string) bool {
	switch value {
	case "party", "candidate":
		return true
	default:
		return false
	}
}
