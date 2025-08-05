package storage

import (
	"log"

	"github.com/dgraph-io/badger/v4"
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

func (bs *BlockStore) Put(data []byte) (string, error) {
	cid := Hash(data)
	return cid, bs.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(cid), data)
	})
}

func (bs *BlockStore) Get(cid string) ([]byte, error) {
	var blockData []byte
	err := bs.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(cid))
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

func (bs *BlockStore) Has(cid string) (bool, error) {
	err := bs.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(cid))
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
