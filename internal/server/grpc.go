package server

import (
	"context"
	"errors"

	"url-shortener/internal/service"
	pb "url-shortener/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCServer struct {
	pb.UnimplementedURLShortenerServer
	svc *service.URLService
}

func NewGRPCServer(svc *service.URLService) *GRPCServer {
	return &GRPCServer{svc: svc}
}

func (s *GRPCServer) Shorten(ctx context.Context, req *pb.ShortenRequest) (*pb.ShortenResponse, error) {
	resp, err := s.svc.Shorten(ctx, req.Url, req.ExpiresIn)
	if err != nil {
		return nil, grpcError(err)
	}

	pbResp := &pb.ShortenResponse{ShortUrl: resp.ShortURL}
	if resp.ExpiresAt != nil {
		pbResp.ExpiresAt = *resp.ExpiresAt
	}
	return pbResp, nil
}

func (s *GRPCServer) Resolve(ctx context.Context, req *pb.ResolveRequest) (*pb.ResolveResponse, error) {
	if req.ShortCode == "" {
		return nil, status.Error(codes.InvalidArgument, "short_code is required")
	}

	originalURL, err := s.svc.Resolve(ctx, req.ShortCode)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.ResolveResponse{OriginalUrl: originalURL}, nil
}

func (s *GRPCServer) GetStats(ctx context.Context, req *pb.StatsRequest) (*pb.StatsResponse, error) {
	if req.ShortCode == "" {
		return nil, status.Error(codes.InvalidArgument, "short_code is required")
	}

	stats, err := s.svc.GetStats(ctx, req.ShortCode)
	if err != nil {
		return nil, grpcError(err)
	}

	resp := &pb.StatsResponse{
		ShortCode:   stats.ShortCode,
		OriginalUrl: stats.OriginalURL,
		ClickCount:  int32(stats.ClickCount),
		CreatedAt:   stats.CreatedAt,
	}
	if stats.ExpiresAt != nil {
		resp.ExpiresAt = *stats.ExpiresAt
	}
	return resp, nil
}

func grpcError(err error) error {
	switch {
	case errors.Is(err, service.ErrValidation):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, service.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
