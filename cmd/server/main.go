// cmd/server/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/Yashh56/p2p-storage/api/v1"
	"github.com/Yashh56/p2p-storage/internal/api"
	"github.com/Yashh56/p2p-storage/internal/node"
	"github.com/Yashh56/p2p-storage/internal/storage"
	"google.golang.org/grpc"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, err := storage.NewBlockStore("./db")
	if err != nil {
		log.Fatalf("Failed to create blockstore: %v", err)
	}
	n, err := node.NewNode(ctx, store)
	if err != nil {
		log.Fatalf("Failed to create P2P node: %v", err)
	}
	defer n.Host.Close()

	fmt.Printf("Node is online with PeerId: %s\n", n.Host.ID())
	fmt.Println("Listen addresses:", n.Host.Addrs())

	go func() {
		apiServer := api.NewServer(n)

		grpcServer := grpc.NewServer()

		pb.RegisterStorageServiceServer(grpcServer, apiServer)

		lis, err := net.Listen("tcp", ":50051")

		if err != nil {
			log.Fatalf("Failed to listen on gRPC port: %s\n", err)
		}
		log.Println("gRPC server listening on :50051")

		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server shut down: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\nShutting Down Node...")
}
