package messenger

import (
	"hashgossip/consts"
	"hashgossip/models"
	"hashgossip/storages"
	"hashgossip/transport"

	"github.com/vmihailenco/msgpack"
	"log"
)

type gossiper struct {
	peers storage.PeerStorage
	ch    chan []byte
}

type Gossiper interface {
	StartLoop()
	SendMessage(models.Message) error
}

func NewGossiper(ps storage.PeerStorage) Gossiper {
	return &gossiper{peers: ps, ch: make(chan []byte, 10)}
}

func (g *gossiper) StartLoop() {
	for {
		mb := <-g.ch
		payload := append(consts.PrefMessage, mb...)

		for _, p := range g.peers.List() {
			err := transport.SendPayloadToUDP(p.ToString(), payload)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func (g *gossiper) SendMessage(msg models.Message) error {
	mb, err := msgpack.Marshal(msg)
	if err != nil {
		return err
	}
	g.ch <- mb
	return nil
}
