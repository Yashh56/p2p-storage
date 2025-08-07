package main

import (
	"context"
	"io"
	"log"
	"os"
	"time"

	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Yashh56/p2p-storage/api/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var addCmd = &cobra.Command{
	Use:   "add [filePath]",
	Short: "Adds a file to the P2P Network",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]

		conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))

		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		client := pb.NewStorageServiceClient(conn)

		file, err := os.Open(filePath)

		if err != nil {
			log.Fatalf("Failed to open file: %v", err)
		}
		defer file.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		stream, err := client.AddFile(ctx)
		if err != nil {
			log.Fatalf("failed to create stream: %v\n", err)
		}
		buf := make([]byte, 1024)

		for {
			n, err := file.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("Failed to read chunk: %v", err)
			}
			if err := stream.Send(&pb.AddFileRequest{
				ChunkData: buf[:n],
			}); err != nil {
				log.Fatalf("Failed to send chunks: %v", err)
			}
		}
		res, err := stream.CloseAndRecv()
		if err != nil {
			log.Fatalf("failed to receive response: %v", err)
		}

		log.Printf("File added successfully! Root CID: %s", res.GetRootCid())

	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
