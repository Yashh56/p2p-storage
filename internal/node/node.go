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

	node.setupBlockRequestHandler()
	return node, nil
}

func (n *Node) AddFile(ctx context.Context, r io.Reader) (cid.Cid, error) {
	// 1. Chunk the file data.
	chunks, err := file.Chunk(r)
	if err != nil {
		return cid.Undef, err
	}

	// routingDiscovery := routing.NewRoutingDiscovery(n.dht)

	// 2. Create a slice to hold the actual cid.Cid objects.
	chunkCIDs := make([]cid.Cid, len(chunks))

	// 3. Process each chunk.
	for i, chunkData := range chunks {
		// a. Save the chunk to the BlockStore and get its CID object.
		c, err := n.store.Put(chunkData)
		if err != nil {
			return cid.Undef, err
		}
		chunkCIDs[i] = c // Store the CID object.

		// b. Announce to the network that we have this CID object.
		fmt.Printf("Announcing provider for chunk %d: %s\n", i, c)
		if err := n.dht.Provide(ctx, c, true); err != nil {
			log.Printf("Error Providing chunk %s: %v", c, err)
		}
	}

	// 4. Create the manifest. The manifest needs strings, so we convert here.
	cidStrs := make([]string, len(chunkCIDs))
	for i, c := range chunkCIDs {
		cidStrs[i] = c.String()
	}
	manifest := &api.Manifest{
		BlockCids: cidStrs,
	}

	// 5. Marshal the manifest into bytes.
	manifestData, err := proto.Marshal(manifest)
	if err != nil {
		return cid.Undef, err
	}

	// 6. Save the manifest data as its own block and get its root CID.
	rootCID, err := n.store.Put(manifestData)
	if err != nil {
		return cid.Undef, err
	}

	// 7. Announce the root CID to the network.
	fmt.Printf("Announcing provider for root manifest: %s\n", rootCID)
	if err := n.dht.Provide(ctx, rootCID, true); err != nil {
		log.Printf("Error providing root manifest %s: %v", rootCID, err)
	}

	// 8. Return the final root CID object.
	return rootCID, nil
}

func (n *Node) GetFile(ctx context.Context, rootCIDStr string) (io.Reader, error) {
	fmt.Printf("Searching for providers for root CID: %s\n", rootCIDStr)

	// 1. Decode the incoming root CID string into a cid.Cid object.
	rootCidObj, err := cid.Decode(rootCIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode root CID: %w", err)
	}

	// 2. Find providers for the root manifest CID object.
	peerChan, err := n.dht.FindProviders(ctx, rootCidObj)
	if err != nil {
		return nil, err
	}

	// Look for the first available peer.
	var provider peer.AddrInfo
	for _, p := range peerChan {
		if p.ID == n.Host.ID() {
			continue // Skip self.
		}
		provider = p
		break
	}
	if provider.ID == "" {
		return nil, fmt.Errorf("no providers found for root CID")
	}
	fmt.Printf("Found provider: %s\n", provider.ID)

	// 3. Request the manifest block from the provider using its string CID.
	manifestData, err := n.requestBlock(ctx, provider, rootCIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest block: %w", err)
	}

	manifest := &api.Manifest{}
	if err := proto.Unmarshal(manifestData, manifest); err != nil {
		return nil, err
	}

	// 4. Request all the data chunks in parallel.
	var wg sync.WaitGroup
	chunks := make(map[int][]byte)
	var mtx sync.Mutex

	for i, chunkCIDStr := range manifest.BlockCids {
		wg.Add(1)
		go func(idx int, cStr string) {
			defer wg.Done()
			chunkData, err := n.requestBlock(ctx, provider, cStr)
			if err != nil {
				fmt.Printf("Error getting block %s: %v\n", cStr, err)
				return
			}
			mtx.Lock()
			chunks[idx] = chunkData
			mtx.Unlock()
		}(i, chunkCIDStr)
	}
	wg.Wait()

	// 5. Reassemble the file in the correct order.
	var fileData []byte
	for i := 0; i < len(manifest.BlockCids); i++ {
		// Check if a chunk was missed due to an error.
		if chunks[i] == nil {
			return nil, fmt.Errorf("failed to retrieve chunk %d", i)
		}
		fileData = append(fileData, chunks[i]...)
	}

	return bytes.NewReader(fileData), nil
}

// requestBlock's signature remains the same as it sends the string CID over the wire.
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

// setupBlockRequestHandler is updated to handle the incoming string CID.
func (n *Node) setupBlockRequestHandler() {
	n.Host.SetStreamHandler(p2p.BlockProtocolID, func(s network.Stream) {
		defer s.Close()

		// Read the requested CID string from the stream.
		cidBytes, err := io.ReadAll(s)
		if err != nil {
			log.Printf("Error reading from stream: %v", err)
			return
		}

		// Decode the string into a cid.Cid object.
		c, err := cid.Decode(string(cidBytes))
		if err != nil {
			log.Printf("Error decoding CID from stream: %v", err)
			return
		}

		// Get the block from storage using the cid.Cid object.
		blockData, err := n.store.Get(c)
		if err != nil {
			log.Printf("Error getting block from store: %v", err)
			return
		}

		// Write the block data back to the requester.
		_, err = s.Write(blockData)
		if err != nil {
			log.Printf("Error writing to stream: %v", err)
		}
	})
}
