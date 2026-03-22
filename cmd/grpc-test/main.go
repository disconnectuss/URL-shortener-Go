package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "url-shortener/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:9090",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("connection failed:", err)
	}
	defer conn.Close()

	client := pb.NewURLShortenerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("=== Shorten ===")
	shortenResp, err := client.Shorten(ctx, &pb.ShortenRequest{
		Url:       "https://grpc.io",
		ExpiresIn: "1h",
	})
	if err != nil {
		log.Fatal("Shorten failed:", err)
	}
	fmt.Printf("Short URL: %s\n", shortenResp.ShortUrl)
	fmt.Printf("Expires at: %s\n", shortenResp.ExpiresAt)

	shortURL := shortenResp.ShortUrl
	code := shortURL[len(shortURL)-6:]

	fmt.Println("\n=== Resolve ===")
	resolveResp, err := client.Resolve(ctx, &pb.ResolveRequest{
		ShortCode: code,
	})
	if err != nil {
		log.Fatal("Resolve failed:", err)
	}
	fmt.Printf("Original URL: %s\n", resolveResp.OriginalUrl)

	fmt.Println("\n=== GetStats ===")
	statsResp, err := client.GetStats(ctx, &pb.StatsRequest{
		ShortCode: code,
	})
	if err != nil {
		log.Fatal("GetStats failed:", err)
	}
	fmt.Printf("Code: %s\n", statsResp.ShortCode)
	fmt.Printf("URL: %s\n", statsResp.OriginalUrl)
	fmt.Printf("Clicks: %d\n", statsResp.ClickCount)
	fmt.Printf("Created: %s\n", statsResp.CreatedAt)
	fmt.Printf("Expires: %s\n", statsResp.ExpiresAt)
}
