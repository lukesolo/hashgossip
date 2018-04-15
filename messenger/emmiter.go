package messenger

import (
	"log"
	"math/rand"
	"time"

	"hashgossip/models"
)

func newRandomMessage(n int) (models.Message, error) {
	msgPayload := make([]byte, n)
	rand.Read(msgPayload)

	return models.NewMessage(msgPayload)
}

func StartEmmitingMessages(g Gossiper, n int, invalidFreq int) {
	for n > 0 {
		time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
		n -= 1

		msg, err := newRandomMessage(32)
		if err != nil {
			log.Println(err)
			continue
		}

		if rand.Intn(100) < invalidFreq {
			rand.Read(msg.Payload)
		}

		g.SendMessage(msg)
	}
	log.Println("Finish emmiting")
}
