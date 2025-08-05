package node

import (
	"io"

	api "github.com/Yashh56/p2p-storage/api/v1"
	"github.com/Yashh56/p2p-storage/internal/file"
	"github.com/Yashh56/p2p-storage/internal/storage"
	"google.golang.org/protobuf/proto"
)

type Node struct {
	store *storage.BlockStore
}

func NewNode(store *storage.BlockStore) *Node {
	return &Node{
		store: store,
	}
}

func (n *Node) AddFile(r io.Reader) (string, error) {
	chunks, err := file.Chunk(r)
	if err != nil {
		return "", err
	}
	chunksCIDs := make([]string, len(chunks))

	for i, chunkData := range chunks {
		cid, err := n.store.Put(chunkData)
		if err != nil {
			return "", err
		}
		chunksCIDs[i] = cid
	}

	manifest := &api.Manifest{
		BlockCids: chunksCIDs,
	}

	manifestData, err := proto.Marshal(manifest)
	if err != nil {
		return "", err
	}

	rootCID, err := n.store.Put(manifestData)

	if err != nil {
		return "", err
	}
	return rootCID, nil
}
