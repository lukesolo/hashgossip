package storage

import (
	"bytes"
	"sync"
)

type hashStorage struct {
	hashes [][]byte
	mutex  *sync.Mutex
}

type HashStorage interface {
	Add([]byte) bool
	IsIn([]byte) bool
}

func NewHashStorage() HashStorage {
	return &hashStorage{mutex: &sync.Mutex{}}
}

func (hs *hashStorage) Add(h []byte) bool {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()

	if !hs.IsIn(h) {
		hs.hashes = append(hs.hashes, h)
		return true
	}

	return false
}

func (hs *hashStorage) IsIn(checkHash []byte) bool {
	for _, h := range hs.hashes {
		if bytes.Equal(h, checkHash) {
			return true
		}
	}
	return false
}
