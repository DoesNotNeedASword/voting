package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"Voting/internal/vote/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNoActivePartyBallot = errors.New("active party ballot not found")
	ErrPartyOptionNotFound = errors.New("party option not found in active ballot")
	ErrAlreadyVoted        = errors.New("user already voted in this ballot")
	ErrDuplicateBallot     = errors.New("ballot with this type already exists for election")
	ErrDuplicateOption     = errors.New("option with this source entity already exists in ballot")
	ErrElectionNotFound    = errors.New("election not found")
	ErrBallotNotFound      = errors.New("ballot not found")
)

type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repository struct {
	db DB
}

func New(db DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateElection(ctx context.Context, input model.CreateElectionInput) (model.Election, error) {
	const query = `
		INSERT INTO elections (title, description, type, starts_at, ends_at, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, title, description, type, starts_at, ends_at, status, created_at, updated_at`

	election, err := scanElection(r.db.QueryRow(
		ctx,
		query,
		input.Title,
		input.Description,
		input.Type,
		input.StartsAt,
		input.EndsAt,
		input.Status,
	))
	if err != nil {
		return model.Election{}, fmt.Errorf("create election: %w", err)
	}

	return election, nil
}

func (r *Repository) CreateBallot(ctx context.Context, input model.CreateBallotInput) (model.Ballot, error) {
	const query = `
		INSERT INTO election_ballots (election_id, ballot_type, title, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, election_id, ballot_type, title, description, created_at, updated_at`

	ballot, err := scanBallot(r.db.QueryRow(
		ctx,
		query,
		input.ElectionID,
		input.BallotType,
		input.Title,
		input.Description,
	))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503":
				return model.Ballot{}, ErrElectionNotFound
			case "23505":
				return model.Ballot{}, ErrDuplicateBallot
			}
		}
		return model.Ballot{}, fmt.Errorf("create ballot: %w", err)
	}

	return ballot, nil
}

func (r *Repository) CreateBallotOption(ctx context.Context, input model.CreateBallotOptionInput) (model.BallotOption, error) {
	const query = `
		INSERT INTO ballot_options (
			ballot_id,
			source_entity_id,
			display_name,
			description,
			image_url,
			snapshot_payload,
			position
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
		RETURNING id, ballot_id, source_entity_id, display_name, description, image_url, snapshot_payload, position, created_at`

	option, err := scanBallotOption(r.db.QueryRow(
		ctx,
		query,
		input.BallotID,
		input.SourceEntityID,
		input.DisplayName,
		input.Description,
		input.ImageURL,
		[]byte(input.SnapshotPayload),
		input.Position,
	))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503":
				return model.BallotOption{}, ErrBallotNotFound
			case "23505":
				return model.BallotOption{}, ErrDuplicateOption
			}
		}
		return model.BallotOption{}, fmt.Errorf("create ballot option: %w", err)
	}

	return option, nil
}

func (r *Repository) CreatePartyBallotOption(ctx context.Context, input model.CreatePartyBallotOptionInput) (model.BallotOption, error) {
	const query = `
		WITH ballot AS (
			INSERT INTO election_ballots (election_id, ballot_type, title, description)
			VALUES ($1, 'party', $2, $3)
			ON CONFLICT (election_id, ballot_type)
			DO UPDATE SET updated_at = now()
			RETURNING id
		)
		INSERT INTO ballot_options (
			ballot_id,
			source_entity_id,
			display_name,
			description,
			image_url,
			snapshot_payload,
			position
		)
		SELECT id, $4, $5, $6, $7, $8::jsonb, $9
		FROM ballot
		RETURNING id, ballot_id, source_entity_id, display_name, description, image_url, snapshot_payload, position, created_at`

	option, err := scanBallotOption(r.db.QueryRow(
		ctx,
		query,
		input.ElectionID,
		input.BallotTitle,
		input.BallotDescription,
		input.SourceEntityID,
		input.DisplayName,
		input.Description,
		input.ImageURL,
		[]byte(input.SnapshotPayload),
		input.Position,
	))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503":
				return model.BallotOption{}, ErrElectionNotFound
			case "23505":
				return model.BallotOption{}, ErrDuplicateOption
			}
		}
		return model.BallotOption{}, fmt.Errorf("create party ballot option: %w", err)
	}

	return option, nil
}

func (r *Repository) CreatePartyVote(ctx context.Context, userID int64, partyID int64) (model.Vote, error) {
	ballotID, optionID, err := r.activePartyBallotOption(ctx, partyID)
	if err != nil {
		return model.Vote{}, err
	}

	const query = `
		INSERT INTO votes (ballot_id, ballot_option_id, user_id)
		VALUES ($1, $2, $3)
		RETURNING id, ballot_id, ballot_option_id, user_id, created_at`

	vote, err := scanVote(r.db.QueryRow(ctx, query, ballotID, optionID, userID))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.Vote{}, ErrAlreadyVoted
		}
		return model.Vote{}, fmt.Errorf("create party vote: %w", err)
	}

	return vote, nil
}

func (r *Repository) activePartyBallotOption(ctx context.Context, partyID int64) (int64, int64, error) {
	const query = `
		SELECT b.id, o.id
		FROM elections e
		JOIN election_ballots b ON b.election_id = e.id
		JOIN ballot_options o ON o.ballot_id = b.id
		WHERE e.status = 'active'
			AND now() >= e.starts_at
			AND now() < e.ends_at
			AND b.ballot_type = 'party'
			AND o.source_entity_id = $1
		ORDER BY e.starts_at DESC, b.id DESC
		LIMIT 1`

	var ballotID int64
	var optionID int64
	if err := r.db.QueryRow(ctx, query, partyID).Scan(&ballotID, &optionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if exists, existsErr := r.activePartyBallotExists(ctx); existsErr != nil {
				return 0, 0, existsErr
			} else if exists {
				return 0, 0, ErrPartyOptionNotFound
			}
			return 0, 0, ErrNoActivePartyBallot
		}
		return 0, 0, fmt.Errorf("find active party ballot option: %w", err)
	}

	return ballotID, optionID, nil
}

func (r *Repository) activePartyBallotExists(ctx context.Context) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM elections e
			JOIN election_ballots b ON b.election_id = e.id
			WHERE e.status = 'active'
				AND now() >= e.starts_at
				AND now() < e.ends_at
				AND b.ballot_type = 'party'
		)`

	var exists bool
	if err := r.db.QueryRow(ctx, query).Scan(&exists); err != nil {
		return false, fmt.Errorf("check active party ballot: %w", err)
	}

	return exists, nil
}

func scanVote(row pgx.Row) (model.Vote, error) {
	var vote model.Vote
	if err := row.Scan(
		&vote.ID,
		&vote.BallotID,
		&vote.BallotOptionID,
		&vote.UserID,
		&vote.CreatedAt,
	); err != nil {
		return model.Vote{}, err
	}

	return vote, nil
}

func scanElection(row pgx.Row) (model.Election, error) {
	var election model.Election
	if err := row.Scan(
		&election.ID,
		&election.Title,
		&election.Description,
		&election.Type,
		&election.StartsAt,
		&election.EndsAt,
		&election.Status,
		&election.CreatedAt,
		&election.UpdatedAt,
	); err != nil {
		return model.Election{}, err
	}

	return election, nil
}

func scanBallot(row pgx.Row) (model.Ballot, error) {
	var ballot model.Ballot
	if err := row.Scan(
		&ballot.ID,
		&ballot.ElectionID,
		&ballot.BallotType,
		&ballot.Title,
		&ballot.Description,
		&ballot.CreatedAt,
		&ballot.UpdatedAt,
	); err != nil {
		return model.Ballot{}, err
	}

	return ballot, nil
}

func scanBallotOption(row pgx.Row) (model.BallotOption, error) {
	var option model.BallotOption
	var snapshotPayload []byte
	if err := row.Scan(
		&option.ID,
		&option.BallotID,
		&option.SourceEntityID,
		&option.DisplayName,
		&option.Description,
		&option.ImageURL,
		&snapshotPayload,
		&option.Position,
		&option.CreatedAt,
	); err != nil {
		return model.BallotOption{}, err
	}

	option.SnapshotPayload = json.RawMessage(snapshotPayload)
	return option, nil
}
