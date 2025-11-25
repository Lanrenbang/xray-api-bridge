package xrayapi

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	log_command "github.com/xtls/xray-core/app/log/command"
	proxyman_command "github.com/xtls/xray-core/app/proxyman/command"
	router_command "github.com/xtls/xray-core/app/router/command"
	stats_command "github.com/xtls/xray-core/app/stats/command"
)

// Client holds all the gRPC service clients.
type Client struct {
	conn *grpc.ClientConn
	LogClient log_command.LoggerServiceClient
	HandlerClient proxyman_command.HandlerServiceClient
	RouterClient router_command.RoutingServiceClient
	StatsClient stats_command.StatsServiceClient
}

// NewClient creates a new Xray gRPC client.
func NewClient(ctx context.Context, grpcAddress string) (*Client, error) {
	conn, err := grpc.DialContext(ctx, grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC server: %w", err)
	}

	return &Client{
		conn: conn,
		LogClient: log_command.NewLoggerServiceClient(conn),
		HandlerClient: proxyman_command.NewHandlerServiceClient(conn),
		RouterClient: router_command.NewRoutingServiceClient(conn),
		StatsClient: stats_command.NewStatsServiceClient(conn),
	}, nil
}

// Close closes the gRPC client connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
