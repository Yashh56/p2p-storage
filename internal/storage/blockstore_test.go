package storage

import (
	"os"
	"testing"
)

func TestBlockStore_PutGetHas(t *testing.T) {
	_ = os.RemoveAll("./tmpdb")
	store, err := NewBlockStore("./tmpdb")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	data := []byte("Hello World Buddy!!!")

	cid, err := store.Put(data)
	if err != nil {
		t.Fatal(err)
	}
	ok, _ := store.Has(cid)
	if !ok {
		t.Fatal("Block not found after Put")
	}

	got, err := store.Get(cid)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatalf("expected %s, got %s", data, got)
	}

}
