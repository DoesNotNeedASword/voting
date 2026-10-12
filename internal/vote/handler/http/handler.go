package http

import (
	"context"
	"encoding/json"
	"errors"
	nethttp "net/http"
	"strconv"
	"time"

	"Voting/internal/vote/model"
	voteservice "Voting/internal/vote/service"

	partyv1 "github.com/DoesNotNeedASword/proto-contracts/gen/go/party/v1"
	usersv1 "github.com/DoesNotNeedASword/proto-contracts/gen/go/user/v1"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service     VoteService
	partyClient PartyClient
	userClient  UserClient
}

func New(service VoteService, user UserClient, party PartyClient) *Handler {
	return &Handler{service: service, userClient: user, partyClient: party}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/elections", h.CreateElection)
	r.Post("/elections/{electionID}/ballots", h.CreateBallot)
	r.Post("/elections/{electionID}/party-options/{partyID}", h.CreatePartyBallotOption)
	r.Post("/ballots/{ballotID}/options", h.CreateBallotOption)
	r.Route("/vote", func(r chi.Router) {
		r.Post("/users/{userID}/parties/{partyID}", h.VoteForParty)
	})
}

func (h *Handler) CreateElection(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request createElectionRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid request body")
		return
	}

	election, err := h.service.CreateElection(r.Context(), model.CreateElectionInput{
		Title:       request.Title,
		Description: request.Description,
		Type:        request.Type,
		StartsAt:    request.StartsAt,
		EndsAt:      request.EndsAt,
		Status:      request.Status,
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}

	writeJSON(w, nethttp.StatusCreated, election)
}

func (h *Handler) CreateBallot(w nethttp.ResponseWriter, r *nethttp.Request) {
	electionID, ok := parseURLInt(w, r, "electionID", "invalid election id")
	if !ok {
		return
	}

	var request createBallotRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid request body")
		return
	}

	ballot, err := h.service.CreateBallot(r.Context(), model.CreateBallotInput{
		ElectionID:  electionID,
		BallotType:  request.BallotType,
		Title:       request.Title,
		Description: request.Description,
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}

	writeJSON(w, nethttp.StatusCreated, ballot)
}

func (h *Handler) CreateBallotOption(w nethttp.ResponseWriter, r *nethttp.Request) {
	ballotID, ok := parseURLInt(w, r, "ballotID", "invalid ballot id")
	if !ok {
		return
	}

	var request createBallotOptionRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid request body")
		return
	}

	option, err := h.service.CreateBallotOption(r.Context(), model.CreateBallotOptionInput{
		BallotID:        ballotID,
		SourceEntityID:  request.SourceEntityID,
		DisplayName:     request.DisplayName,
		Description:     request.Description,
		ImageURL:        request.ImageURL,
		SnapshotPayload: request.SnapshotPayload,
		Position:        request.Position,
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}

	writeJSON(w, nethttp.StatusCreated, option)
}

func (h *Handler) CreatePartyBallotOption(w nethttp.ResponseWriter, r *nethttp.Request) {
	electionID, ok := parseURLInt(w, r, "electionID", "invalid election id")
	if !ok {
		return
	}
	partyID, ok := parseURLInt(w, r, "partyID", "invalid party id")
	if !ok {
		return
	}

	var request createPartyBallotOptionRequest
	if r.ContentLength > 0 {
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, nethttp.StatusBadRequest, "invalid request body")
			return
		}
	}

	party, err := h.partyClient.GetParty(r.Context(), partyID)
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, err.Error())
		return
	}

	snapshotPayload, err := json.Marshal(map[string]any{
		"id":                party.GetId(),
		"name":              party.GetName(),
		"description":       party.GetDescription(),
		"promises":          party.GetPromises(),
		"image_url":         party.GetImageUrl(),
		"moderation_status": party.GetModerationStatus().String(),
		"moderation_reason": party.GetModerationReason(),
	})
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, err.Error())
		return
	}

	option, err := h.service.CreatePartyBallotOption(r.Context(), model.CreatePartyBallotOptionInput{
		ElectionID:        electionID,
		SourceEntityID:    party.GetId(),
		DisplayName:       party.GetName(),
		Description:       party.GetDescription(),
		ImageURL:          party.GetImageUrl(),
		SnapshotPayload:   snapshotPayload,
		Position:          request.Position,
		BallotTitle:       request.BallotTitle,
		BallotDescription: request.BallotDescription,
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}

	writeJSON(w, nethttp.StatusCreated, option)
}

func (h *Handler) VoteForParty(w nethttp.ResponseWriter, r *nethttp.Request) {
	partyID, err := strconv.ParseInt(chi.URLParam(r, "partyID"), 10, 64)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid party id")
		return
	}
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid user id")
		return
	}

	if _, err := h.userClient.GetUser(r.Context(), userID); err != nil {
		writeError(w, nethttp.StatusInternalServerError, err.Error())
		return
	}
	if _, err := h.partyClient.GetParty(r.Context(), partyID); err != nil {
		writeError(w, nethttp.StatusInternalServerError, err.Error())
		return
	}

	vote, err := h.service.VoteForParty(r.Context(), userID, partyID)
	if err != nil {
		writeVoteError(w, err)
		return
	}

	writeJSON(w, nethttp.StatusCreated, vote)
}

func parseURLInt(w nethttp.ResponseWriter, r *nethttp.Request, name string, message string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, nethttp.StatusBadRequest, message)
		return 0, false
	}

	return id, true
}

func parseID(w nethttp.ResponseWriter, r *nethttp.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, nethttp.StatusBadRequest, "invalid party id")
		return 0, false
	}

	return id, true
}

func intQuery(r *nethttp.Request, name string, defaultValue int) (int, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue, nil
	}

	return strconv.Atoi(value)
}

func decodeJSON(r *nethttp.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeError(w nethttp.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w nethttp.ResponseWriter, status int, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

type errorResponse struct {
	Error string `json:"error"`
}
type UserClient interface {
	GetUser(ctx context.Context, id int64) (*usersv1.User, error)
}
type PartyClient interface {
	GetParty(ctx context.Context, id int64) (*partyv1.Party, error)
}
type VoteService interface {
	CreateElection(ctx context.Context, input model.CreateElectionInput) (model.Election, error)
	CreateBallot(ctx context.Context, input model.CreateBallotInput) (model.Ballot, error)
	CreateBallotOption(ctx context.Context, input model.CreateBallotOptionInput) (model.BallotOption, error)
	CreatePartyBallotOption(ctx context.Context, input model.CreatePartyBallotOptionInput) (model.BallotOption, error)
	VoteForParty(ctx context.Context, userID int64, partyID int64) (model.Vote, error)
}

type createElectionRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	Status      string    `json:"status"`
}

type createBallotRequest struct {
	BallotType  string `json:"ballot_type"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type createBallotOptionRequest struct {
	SourceEntityID  int64           `json:"source_entity_id"`
	DisplayName     string          `json:"display_name"`
	Description     string          `json:"description"`
	ImageURL        string          `json:"image_url"`
	SnapshotPayload json.RawMessage `json:"snapshot_payload"`
	Position        int32           `json:"position"`
}

type createPartyBallotOptionRequest struct {
	Position          int32  `json:"position"`
	BallotTitle       string `json:"ballot_title"`
	BallotDescription string `json:"ballot_description"`
}

func writeVoteError(w nethttp.ResponseWriter, err error) {
	switch {
	case errors.Is(err, voteservice.ErrInvalidInput):
		writeError(w, nethttp.StatusBadRequest, err.Error())
	case voteservice.IsAlreadyVoted(err):
		writeError(w, nethttp.StatusConflict, err.Error())
	case voteservice.IsNoActivePartyBallot(err), voteservice.IsPartyOptionNotFound(err):
		writeError(w, nethttp.StatusNotFound, err.Error())
	default:
		writeError(w, nethttp.StatusInternalServerError, err.Error())
	}
}

func writeCommandError(w nethttp.ResponseWriter, err error) {
	switch {
	case errors.Is(err, voteservice.ErrInvalidInput):
		writeError(w, nethttp.StatusBadRequest, err.Error())
	case voteservice.IsDuplicateBallot(err), voteservice.IsDuplicateOption(err):
		writeError(w, nethttp.StatusConflict, err.Error())
	case voteservice.IsElectionNotFound(err), voteservice.IsBallotNotFound(err):
		writeError(w, nethttp.StatusNotFound, err.Error())
	default:
		writeError(w, nethttp.StatusInternalServerError, err.Error())
	}
}
