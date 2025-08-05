// cmd/server/main.go
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Yashh56/p2p-storage/internal/node"
	"github.com/Yashh56/p2p-storage/internal/storage"
)

func main() {
	dbPath := "./db"
	defer os.RemoveAll(dbPath)

	store, err := storage.NewBlockStore(dbPath)
	if err != nil {
		log.Fatal(err)
	}

	n := node.NewNode(store)
	dummyData := "This is a test file for our p2p storage. Currently in the testing Mode and learning Mode."
	reader := strings.NewReader(dummyData)

	rootCID, err := n.AddFile(reader)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Successfully added file. Root CID : %s\n", rootCID)
}
