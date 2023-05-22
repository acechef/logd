package server

import (
	"context"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	api "logd/api/v1"
	"logd/internal/config"
	"logd/internal/log"
	"net"
	"os"
	"testing"
)

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T, client api.LogClient, config *Config){
		"produce/consume a message to/from the succeeds": testProduceConsume,
		"produce/consume stream succeeds":                testProduceConsumeStream,
		"consume past log boundary fails":                testConsumePastBoundary,
	} {
		t.Run(scenario, func(t *testing.T) {
			client, config, teardown := setupTest(t, nil)
			defer teardown()
			fn(t, client, config)
		})
	}
}

func setupTest(t *testing.T, fn func(config *Config)) (api.LogClient, *Config, func()) {
	t.Helper()

	listen, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	clientTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CAFile: config.CAFile,
	})
	require.NoError(t, err)

	clientCreds := credentials.NewTLS(clientTLSConfig)

	//clientOptions := []grpc.DialOption{grpc.WithInsecure()}
	//conn, err := grpc.Dial(listen.Addr().String(), clientOptions...)
	conn, err := grpc.Dial(listen.Addr().String(), grpc.WithTransportCredentials(clientCreds))
	require.NoError(t, err)

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		ServerAddress: listen.Addr().String(),
	})
	require.NoError(t, err)
	serverCreds := credentials.NewTLS(serverTLSConfig)

	dir, err := os.MkdirTemp("", "server-test")
	require.NoError(t, err)

	cLog, err := log.NewLog(dir, log.Config{})
	require.NoError(t, err)

	cfg := &Config{CommitLog: cLog}
	if fn != nil {
		fn(cfg)
	}
	server, err := NewGRPCServer(cfg, grpc.Creds(serverCreds))
	require.NoError(t, err)

	go func() {
		server.Serve(listen)
	}()

	client := api.NewLogClient(conn)

	return client, cfg, func() {
		server.Stop()
		conn.Close()
		listen.Close()
		cLog.Close()
	}
}

func testProduceConsume(t *testing.T, client api.LogClient, config *Config) {
	ctx := context.Background()
	want := &api.Record{
		Value: []byte("hello world"),
	}
	produceResponse, err := client.Produce(ctx, &api.ProduceRequest{Record: want})
	require.NoError(t, err)

	consumeResponse, err := client.Consume(ctx, &api.ConsumeRequest{Offset: produceResponse.Offset})
	require.NoError(t, err)
	require.Equal(t, want.Value, consumeResponse.Record.Value)
	require.Equal(t, want.Offset, consumeResponse.Record.Offset)
}

func testProduceConsumeStream(t *testing.T, client api.LogClient, config *Config) {
	ctx := context.Background()
	records := []*api.Record{{Value: []byte("first message"), Offset: 0}, {Value: []byte("second message"), Offset: 1}}
	{
		stream, err := client.ProduceStream(ctx)
		require.NoError(t, err)
		for offset, record := range records {
			err = stream.Send(&api.ProduceRequest{Record: record})
			require.NoError(t, err)
			recv, err := stream.Recv()
			require.NoError(t, err)
			if recv.Offset != uint64(offset) {
				t.Fatalf("got offset: %d, want: %d", recv.Offset, offset)
			}
		}
	}
	{
		stream, err := client.ConsumeStream(ctx, &api.ConsumeRequest{Offset: 0})
		require.NoError(t, err)

		for i, record := range records {
			recv, err := stream.Recv()
			require.NoError(t, err)
			require.Equal(t, recv.Record, &api.Record{
				Value:  record.Value,
				Offset: uint64(i),
			})
		}
	}
}

func testConsumePastBoundary(t *testing.T, client api.LogClient, config *Config) {
	ctx := context.Background()
	produce, err := client.Produce(ctx, &api.ProduceRequest{
		Record: &api.Record{Value: []byte("hello world")},
	})
	require.NoError(t, err)

	consume, err := client.Consume(ctx, &api.ConsumeRequest{Offset: produce.Offset + 1})
	if consume != nil {
		t.Fatal("consume not nil")
	}
	got := status.Code(err)
	want := status.Code(api.ErrOffsetOutOfRange{}.GRPCStatus().Err())
	if got != want {
		t.Fatalf("got err: %v, want: %v", got, want)
	}
}
