package storage

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

func Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

var DefaultCidBuilder = cid.V1Builder{
	Codec:    cid.DagProtobuf,
	MhType:   multihash.SHA2_256,
	MhLength: -1, // Default length for the hash type
}

// Sum creates a new CID by hashing the given data.
func Sum(data []byte) (cid.Cid, error) {
	return DefaultCidBuilder.Sum(data)
}
