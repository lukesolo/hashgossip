package storage

import (
	"log"
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
		log.Printf("new message was set %+v", ms.msg.GetPayload()[0:5])
		return true
	}
	return false
}

func (ms *messageStorage) Get() models.Message {
	return ms.msg
}
