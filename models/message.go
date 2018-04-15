package models

import (
	"bytes"
	"crypto/sha1"
	"log"
)

func calcChecksum(b []byte) ([]byte, error) {
	hasher := sha1.New()
	_, err := hasher.Write(b)
	if err != nil {
		return []byte{}, err
	}

	return hasher.Sum(nil), nil
}

type Message struct {
	Payload  []byte
	Checksum []byte
}

func NewMessage(payload []byte) (Message, error) {
	msgChecksum, err := calcChecksum(payload)
	if err != nil {
		return Message{}, err
	}

	return Message{Payload: payload, Checksum: msgChecksum}, nil
}

func (m Message) IsValid() bool {
	msgChecksum, err := calcChecksum(m.Payload)
	if err != nil {
		log.Println(err)
		return false
	}

	return bytes.Equal(m.Checksum, msgChecksum)
}

func (m Message) Compare(b Message) int {
	return bytes.Compare(m.Checksum, b.Checksum)
}

func (m Message) GetPayload() []byte {
	return m.Payload
}

func (m Message) GetHash() []byte {
	return m.Checksum
}

func (m Message) IsEmpty() bool {
	return len(m.Payload) == 0
}
