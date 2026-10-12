package party

import (
	"context"
	"errors"
	"fmt"
	"time"

	partiesv1 "github.com/DoesNotNeedASword/proto-contracts/gen/go/party/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	api  partiesv1.PartyServiceClient
	conn *grpc.ClientConn
}

func New(target string) (*Client, error) {
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}

	return &Client{
		api:  partiesv1.NewPartyServiceClient(conn),
		conn: conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetParty(ctx context.Context, id int64) (*partiesv1.Party, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := c.api.GetParty(ctx, &partiesv1.GetPartyRequest{Id: id})
	if err != nil {
		return nil, fmt.Errorf("get party: %w", err)
	}
	if resp.GetParty() == nil {
		return nil, errors.New("get party: empty party response")
	}
	return resp.GetParty(), nil
}
