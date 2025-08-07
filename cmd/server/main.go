// cmd/server/main.go
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Yashh56/p2p-storage/internal/node"
	"github.com/Yashh56/p2p-storage/internal/storage"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

func main() {
	// Define command-line flags
	dest := flag.String("dest", "", "Destination multiaddr string")
	getCID := flag.String("getCID", "", "CID of the file to retrieve")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbPath := fmt.Sprintf("./db_%d", time.Now().UnixNano())
	store, err := storage.NewBlockStore(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dbPath)

	n, err := node.NewNode(ctx, store)
	if err != nil {
		log.Fatal(err)
	}
	defer n.Host.Close()

	fmt.Printf("Node is online with PeerID: %s\n", n.Host.ID())
	fmt.Println("Listen addresses:", n.Host.Addrs())

	// If a destination is provided, connect to it
	if *dest != "" {
		maddr, err := multiaddr.NewMultiaddr(*dest)
		if err != nil {
			log.Fatal(err)
		}
		addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			log.Fatal(err)
		}
		if err := n.Host.Connect(ctx, *addrInfo); err != nil {
			log.Fatalf("Failed to connect to destination peer: %v", err)
		}
		fmt.Printf("Successfully connected to destination peer: %s\n", *dest)

		fmt.Println("Connection established. Waiting for DHT to settle...")
		time.Sleep(time.Second * 5)
	}

	// If getCID is provided, act as a "leecher"
	if *getCID != "" {
		fmt.Println("\n--- GETTING FILE ---")
		reader, err := n.GetFile(ctx, *getCID)
		if err != nil {
			log.Fatalf("Failed to get file: %v", err)
		}
		content, err := io.ReadAll(reader)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Successfully retrieved file! Content:\n\n%s\n", string(content))
	} else {
		// Otherwise, act as a "seeder"
		fmt.Println("\n--- ADDING FILE ---")
		// Give the network a moment to stabilize before adding the file
		time.Sleep(2 * time.Second)
		testData := "Hello from the P2P network! This is a test."
		reader := bytes.NewReader([]byte(testData))
		rootCID, err := n.AddFile(ctx, reader)
		if err != nil {
			log.Fatalf("Failed to add file: %v", err)
		}
		fmt.Printf("✅ File added successfully! Root CID: %s\n", rootCID.String())
		fmt.Println("This node is now seeding the file. Keep it running.")
	}

	// Wait for a shutdown signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	fmt.Println("Shutting down node...")
}
