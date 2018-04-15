package storage

import (
	"sync"

	"hashgossip/models"
)

type messageStorage struct {
	msg   models.Message
	mutex *sync.Mutex
}

type MessageStorage interface {
	Set(models.Message) bool
	Get() models.Message
}

func NewMessageStorage() MessageStorage {
	return &messageStorage{mutex: &sync.Mutex{}}
}

func (ms *messageStorage) Set(m models.Message) bool {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	if ms.msg.Compare(m) < 0 {
		ms.msg = m
		return true
	}
	return false
}

func (ms *messageStorage) Get() models.Message {
	return ms.msg
}
