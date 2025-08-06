// cmd/server/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Yashh56/p2p-storage/internal/node"
	"github.com/Yashh56/p2p-storage/internal/storage"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbPath := "./db"
	defer os.RemoveAll(dbPath)

	store, err := storage.NewBlockStore(dbPath)
	if err != nil {
		log.Fatal(err)
	}

	n, err := node.NewNode(ctx, store)

	if err != nil {
		log.Fatal(err)
	}
	defer n.Host.Close()

	fmt.Println("Node is Online. Press Ctrl + C to Shut Down.")

	dummyData := "This is a test file for our p2p storage. Currently in the testing Mode and learning Mode."
	reader := strings.NewReader(dummyData)

	rootCID, err := n.AddFile(reader)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Successfully added file. Root CID : %s\n", rootCID)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("Shutting Down the Node")

}
