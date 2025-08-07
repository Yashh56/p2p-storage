package api

import (
	"io"
	"log"

	api "github.com/Yashh56/p2p-storage/api/v1"
	"github.com/Yashh56/p2p-storage/internal/node"
)

type Server struct {
	api.UnimplementedStorageServiceServer
	node *node.Node
}

func NewServer(node *node.Node) *Server {
	return &Server{
		node: node,
	}
}

func (s *Server) AddFile(stream api.StorageService_AddFileServer) error {
	log.Println("Received AddFile Request")

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("Error receiving from stream: %v", err)
				return
			}

			if _, err := pw.Write(req.GetChunkData()); err != nil {
				log.Printf("Error Writing to Pipe: %v", err)
				return
			}
		}
	}()

	rootCID, err := s.node.AddFile(stream.Context(), pr)
	if err != nil {
		return err
	}
	return stream.SendAndClose(&api.AddFileResponse{
		RootCid: rootCID.String(),
	})
}

func (s *Server) GetFile(req *api.GetFileRequest, stream api.StorageService_GetFileServer) error {
	log.Printf("Received GetFile request for CID: %s", req.GetCid())
	reader, err := s.node.GetFile(stream.Context(), req.GetCid())
	if err != nil {
		return err
	}

	buf := make([]byte, 1024*64)

	for {
		n, err := reader.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading file data: %v", err)
			return err
		}

		// Send the chunk to the client via the stream.
		if err := stream.Send(&api.GetFileResponse{ChunkData: buf[:n]}); err != nil {
			log.Printf("Error sending data to stream: %v", err)
			return err
		}
	}

	log.Println("Finished streaming file.")
	return nil
}
