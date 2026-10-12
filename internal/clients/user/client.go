package userclient

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	usersv1 "github.com/DoesNotNeedASword/proto-contracts/gen/go/user/v1"
)

type Client struct {
	api  usersv1.UserServiceClient
	conn *grpc.ClientConn
}

func New(target string) (*Client, error) {
	conn, err := grpc.NewClient(
		target, // например "users-service:50051"
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// для production — TLS:
		// grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{...})),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}

	return &Client{
		api:  usersv1.NewUserServiceClient(conn),
		conn: conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// Обёртка над методом — удобно для handler'ов
func (c *Client) GetUser(ctx context.Context, id int64) (*usersv1.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := c.api.GetUser(ctx, &usersv1.GetUserRequest{Id: id})
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return resp.GetUser(), nil
}
