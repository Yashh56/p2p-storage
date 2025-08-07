package node

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"sync"

	api "github.com/Yashh56/p2p-storage/api/v1"
	"github.com/Yashh56/p2p-storage/internal/file"
	"github.com/Yashh56/p2p-storage/internal/p2p"
	"github.com/Yashh56/p2p-storage/internal/storage"
	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/net/context"
	"google.golang.org/protobuf/proto"
)

type Node struct {
	store *storage.BlockStore
	Host  host.Host
	dht   *dht.IpfsDHT
}

// NewNode creates a new P2P node.
func NewNode(ctx context.Context, store *storage.BlockStore) (*Node, error) {
	h, err := p2p.NewHost(ctx)
	if err != nil {
		return nil, err
	}

	dht, err := p2p.InitDHT(ctx, h)
	if err != nil {
		return nil, err
	}

	node := &Node{
		store: store,
		Host:  h,
		dht:   dht,
	}

	// Register the handler that allows this node to respond to block requests.
	node.setupBlockRequestHandler()

	return node, nil
}

// AddFile chunks a file, stores it locally, and announces it to the network.
func (n *Node) AddFile(ctx context.Context, r io.Reader) (cid.Cid, error) {
	chunks, err := file.Chunk(r)
	if err != nil {
		return cid.Undef, err
	}

	chunkCIDs := make([]cid.Cid, len(chunks))
	for i, chunkData := range chunks {
		c, err := n.store.Put(chunkData)
		if err != nil {
			return cid.Undef, err
		}
		chunkCIDs[i] = c

		fmt.Printf("Announcing provider for chunk %d: %s\n", i, c)
		if err := n.dht.Provide(ctx, c, true); err != nil {
			log.Printf("Error providing chunk %s: %v", c, err)
		}
	}

	cidStrs := make([]string, len(chunkCIDs))
	for i, c := range chunkCIDs {
		cidStrs[i] = c.String()
	}
	manifest := &api.Manifest{BlockCids: cidStrs}

	manifestData, err := proto.Marshal(manifest)
	if err != nil {
		return cid.Undef, err
	}

	rootCID, err := n.store.Put(manifestData)
	if err != nil {
		return cid.Undef, err
	}

	fmt.Printf("Announcing provider for root manifest: %s\n", rootCID)
	if err := n.dht.Provide(ctx, rootCID, true); err != nil {
		log.Printf("Error providing root manifest %s: %v", rootCID, err)
	}

	return rootCID, nil
}

// GetFile retrieves a file. It checks the local store first, then searches the network.
func (n *Node) GetFile(ctx context.Context, rootCIDStr string) (io.Reader, error) {
	log.Printf("Attempting to get file with root CID: %s", rootCIDStr)

	rootCidObj, err := cid.Decode(rootCIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode root CID: %w", err)
	}

	// --- THE COMPLETE FIX ---
	// 1. Check if we have the root manifest locally.
	manifestData, err := n.store.Get(rootCidObj)
	if err == nil {
		// LOCAL PATH: We have the manifest. Assume all chunks are local.
		log.Println("Content found locally. Retrieving from disk.")
		return n.retrieveFileFromLocalStore(manifestData)
	}

	// NETWORK PATH: We don't have it locally, so search the network.
	log.Println("Content not found locally, searching network...")
	return n.retrieveFileFromNetwork(ctx, rootCidObj)
}

// retrieveFileFromLocalStore is called when the root manifest is already in our blockstore.
func (n *Node) retrieveFileFromLocalStore(manifestData []byte) (io.Reader, error) {
	manifest := &api.Manifest{}
	if err := proto.Unmarshal(manifestData, manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal local manifest: %w", err)
	}

	var fileData []byte
	for _, chunkCIDStr := range manifest.BlockCids {
		c, err := cid.Decode(chunkCIDStr)
		if err != nil {
			return nil, err
		}
		chunkData, err := n.store.Get(c)
		if err != nil {
			return nil, fmt.Errorf("failed to get local chunk %s: %w", c, err)
		}
		fileData = append(fileData, chunkData...)
	}
	return bytes.NewReader(fileData), nil
}

// retrieveFileFromNetwork finds providers and fetches the file block by block.
func (n *Node) retrieveFileFromNetwork(ctx context.Context, rootCidObj cid.Cid) (io.Reader, error) {
	peerChan, err := n.dht.FindProviders(ctx, rootCidObj)
	if err != nil {
		return nil, err
	}

	var provider peer.AddrInfo
	for _, p := range peerChan {
		if p.ID == n.Host.ID() {
			continue
		}
		provider = p
		break
	}
	if provider.ID == "" {
		return nil, fmt.Errorf("no providers found for root CID")
	}
	log.Printf("Found provider: %s", provider.ID)

	manifestData, err := n.requestBlock(ctx, provider, rootCidObj.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest block from network: %w", err)
	}

	manifest := &api.Manifest{}
	if err := proto.Unmarshal(manifestData, manifest); err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	chunks := make(map[int][]byte)
	var mtx sync.Mutex

	for i, chunkCIDStr := range manifest.BlockCids {
		wg.Add(1)
		go func(idx int, cStr string) {
			defer wg.Done()
			chunkData, err := n.requestBlock(ctx, provider, cStr)
			if err != nil {
				log.Printf("Error getting block %s: %v", cStr, err)
				return
			}
			mtx.Lock()
			chunks[idx] = chunkData
			mtx.Unlock()
		}(i, chunkCIDStr)
	}
	wg.Wait()

	var fileData []byte
	for i := 0; i < len(manifest.BlockCids); i++ {
		if chunks[i] == nil {
			return nil, fmt.Errorf("failed to retrieve chunk %d", i)
		}
		fileData = append(fileData, chunks[i]...)
	}

	return bytes.NewReader(fileData), nil
}

// requestBlock handles sending a request for a block to a peer.
func (n *Node) requestBlock(ctx context.Context, p peer.AddrInfo, cidStr string) ([]byte, error) {
	s, err := n.Host.NewStream(ctx, p.ID, p2p.BlockProtocolID)
	if err != nil {
		return nil, err
	}
	defer s.Close()

	_, err = s.Write([]byte(cidStr))
	if err != nil {
		return nil, err
	}
	s.CloseWrite()

	return io.ReadAll(s)
}

// setupBlockRequestHandler sets up the handler for responding to block requests.
func (n *Node) setupBlockRequestHandler() {
	n.Host.SetStreamHandler(p2p.BlockProtocolID, func(s network.Stream) {
		defer s.Close()
		cidBytes, err := io.ReadAll(s)
		if err != nil {
			log.Printf("Error reading from stream: %v", err)
			return
		}
		c, err := cid.Decode(string(cidBytes))
		if err != nil {
			log.Printf("Error decoding CID from stream: %v", err)
			return
		}
		blockData, err := n.store.Get(c)
		if err != nil {
			log.Printf("Error getting block from store: %v", err)
			return
		}
		_, err = s.Write(blockData)
		if err != nil {
			log.Printf("Error writing to stream: %v", err)
		}
	})
}
