package storage

import (
	"log"

	"github.com/dgraph-io/badger/v4"
	"github.com/ipfs/go-cid"
)

type BlockStore struct {
	db *badger.DB
}

func NewBlockStore(path string) (*BlockStore, error) {
	opts := badger.DefaultOptions(path)
	db, err := badger.Open(opts)

	if err != nil {
		return nil, err
	}
	return &BlockStore{
		db: db,
	}, nil
}

func (bs *BlockStore) Put(data []byte) (cid.Cid, error) {
	c, err := Sum(data)
	if err != nil {
		return cid.Undef, err
	}
	key := []byte(c.KeyString())
	return c, bs.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, data)
	})
}

func (bs *BlockStore) Get(c cid.Cid) ([]byte, error) {
	var blockData []byte
	key := []byte(c.KeyString())
	err := bs.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			blockData = append([]byte{}, val...)
			return nil
		})
	})
	return blockData, err
}

func (bs *BlockStore) Has(c cid.Cid) (bool, error) {
	key := []byte(c.KeyString())
	err := bs.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		return err
	})
	return err == nil, err
}

func (bs *BlockStore) Close() {
	err := bs.db.Close()
	if err != nil {
		log.Println("Error Closing the BadgerDB", err)
	}
}
