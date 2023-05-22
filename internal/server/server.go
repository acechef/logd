package server

import (
	"context"
	"google.golang.org/grpc"
	api "logd/api/v1"
)

type Config struct {
	CommitLog CommitLog
}

var _ api.LogServer = (*grpcServer)(nil)

type grpcServer struct {
	api.UnimplementedLogServer
	*Config
}

func NewGrpcServer(config *Config) (*grpcServer, error) {
	srv := &grpcServer{
		Config: config,
	}
	return srv, nil
}

func NewGRPCServer(config *Config, option ...grpc.ServerOption) (*grpc.Server, error) {
	server := grpc.NewServer(option...)
	srv, err := NewGrpcServer(config)
	if err != nil {
		return nil, err
	}
	api.RegisterLogServer(server, srv)
	return server, nil
}
func (s *grpcServer) Produce(ctx context.Context, req *api.ProduceRequest) (*api.ProduceResponse, error) {
	offset, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &api.ProduceResponse{Offset: offset}, nil
}

func (s *grpcServer) Consume(ctx context.Context, req *api.ConsumeRequest) (*api.ConsumeResponse, error) {
	record, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		return nil, err
	}
	return &api.ConsumeResponse{Record: record}, nil
}

func (s *grpcServer) ProduceStream(stream api.Log_ProduceStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}
		res, err := s.Produce(stream.Context(), req)
		if err != nil {
			return err
		}
		if err := stream.Send(res); err != nil {
			return err
		}
	}
}

func (s *grpcServer) ConsumeStream(request *api.ConsumeRequest, stream api.Log_ConsumeStreamServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			res, err := s.Consume(stream.Context(), request)
			switch err.(type) {
			case nil:
			case api.ErrOffsetOutOfRange:
				continue
			default:
				return err
			}
			if err := stream.Send(res); err != nil {
				return err
			}
			request.Offset++
		}
	}
}

type CommitLog interface {
	Append(*api.Record) (uint64, error)
	Read(uint64) (*api.Record, error)
}
