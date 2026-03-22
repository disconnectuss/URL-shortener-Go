package main

import (
	"context"
	"fmt"
	"time"

	pb "url-shortener/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcServer struct {
	pb.UnimplementedURLShortenerServer
	store   *URLStore
	baseURL string
}

func newGRPCServer(store *URLStore, baseURL string) *grpcServer {
	return &grpcServer{
		store:   store,
		baseURL: baseURL,
	}
}

func (s *grpcServer) Shorten(ctx context.Context, req *pb.ShortenRequest) (*pb.ShortenResponse, error) {
	if req.Url == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	var expiresAt *time.Time
	if req.ExpiresIn != "" {
		d, err := parseDuration(req.ExpiresIn)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid expires_in")
		}
		t := time.Now().Add(d)
		expiresAt = &t
	}

	code, err := generateShortCode()
	if err != nil {
		return nil, status.Error(codes.Internal, "could not generate short code")
	}

	if err := s.store.Save(code, req.Url, expiresAt); err != nil {
		return nil, status.Error(codes.Internal, "could not save URL")
	}

	resp := &pb.ShortenResponse{
		ShortUrl: fmt.Sprintf("%s/%s", s.baseURL, code),
	}
	if expiresAt != nil {
		resp.ExpiresAt = expiresAt.UTC().Format(time.RFC3339)
	}

	return resp, nil
}

func (s *grpcServer) Resolve(ctx context.Context, req *pb.ResolveRequest) (*pb.ResolveResponse, error) {
	if req.ShortCode == "" {
		return nil, status.Error(codes.InvalidArgument, "short_code is required")
	}

	originalURL, err := s.store.Get(req.ShortCode)
	if err != nil {
		return nil, status.Error(codes.NotFound, "URL not found")
	}

	s.store.IncrementClick(req.ShortCode)

	return &pb.ResolveResponse{OriginalUrl: originalURL}, nil
}

func (s *grpcServer) GetStats(ctx context.Context, req *pb.StatsRequest) (*pb.StatsResponse, error) {
	if req.ShortCode == "" {
		return nil, status.Error(codes.InvalidArgument, "short_code is required")
	}

	stats, err := s.store.GetStats(req.ShortCode)
	if err != nil {
		return nil, status.Error(codes.NotFound, "URL not found")
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
