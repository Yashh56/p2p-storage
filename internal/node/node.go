package node

import (
	"io"

	api "github.com/Yashh56/p2p-storage/api/v1"
	"github.com/Yashh56/p2p-storage/internal/file"
	"github.com/Yashh56/p2p-storage/internal/p2p"
	"github.com/Yashh56/p2p-storage/internal/storage"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"golang.org/x/net/context"
	"google.golang.org/protobuf/proto"
)

type Node struct {
	store *storage.BlockStore
	Host  host.Host
	dht   *dht.IpfsDHT
}

func NewNode(ctx context.Context, store *storage.BlockStore) (*Node, error) {
	h, err := p2p.NewHost(ctx)
	if err != nil {
		return nil, err
	}
	dht, err := p2p.InitDHT(ctx, h)
	if err != nil {
		return nil, err
	}
	return &Node{
		store: store,
		Host:  h,
		dht:   dht,
	}, nil
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
