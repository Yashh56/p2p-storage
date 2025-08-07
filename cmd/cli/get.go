// cmd/cli/get.go
package main

import (
	"context"
	"io"
	"log"
	"os"
	"time"

	pb "github.com/Yashh56/p2p-storage/api/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var getCmd = &cobra.Command{
	Use:   "get [cid] [output_filepath]",
	Short: "Retrieves a file from the P2P network using its CID",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		cid := args[0]
		outputFilepath := args[1]

		// 1. Connect to the gRPC server.
		conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		client := pb.NewStorageServiceClient(conn)

		// 2. Create the output file.
		file, err := os.Create(outputFilepath)
		if err != nil {
			log.Fatalf("failed to create output file: %v", err)
		}
		defer file.Close()

		// 3. Call the GetFile RPC.
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1) // 1 minute timeout
		defer cancel()
		stream, err := client.GetFile(ctx, &pb.GetFileRequest{Cid: cid})
		if err != nil {
			log.Fatalf("failed to call GetFile: %v", err)
		}

		// 4. Receive the file in a stream of chunks.
		log.Println("Receiving file...")
		for {
			res, err := stream.Recv()
			// io.EOF means the stream has finished successfully.
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("failed to receive chunk: %v", err)
			}

			// Write the received chunk to the output file.
			if _, err := file.Write(res.GetChunkData()); err != nil {
				log.Fatalf("failed to write chunk to file: %v", err)
			}
		}

		log.Printf("File successfully retrieved and saved to: %s", outputFilepath)
	},
}

// init registers the get command with the root command.
func init() {
	rootCmd.AddCommand(getCmd)
}
