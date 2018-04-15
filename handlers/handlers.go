package handlers

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"os"

	c "hashgossip/consts"
	"hashgossip/messenger"
	"hashgossip/models"
	"hashgossip/storages"
	"hashgossip/transport"

	"github.com/vmihailenco/msgpack"
)

type UdpHandler struct {
	PeerStorage    storage.PeerStorage
	MessageStorage storage.MessageStorage
	HashStorage    storage.HashStorage
	Gossiper       messenger.Gossiper
}

func (u UdpHandler) Handler(src *net.UDPAddr, n int, buf []byte) {
	header, body := buf[:c.PrefLen], buf[c.PrefLen:n]
	switch {
	case bytes.Equal(header, c.PrefMessage):
		u.messageHandler(src, body)
	case bytes.Equal(header, c.PrefWelcome):
		u.welcomeHandler(src, body)
	case bytes.Equal(header, c.PrefReport):
		u.reportHandler(src, body)
	case bytes.Equal(header, c.PrefMonitoring):
		u.monitoringHandler(src, body)
	case bytes.Equal(header, c.PrefShutdown):
		os.Exit(2)
	case bytes.Equal(header, c.PrefHello):
		u.helloHandler(src, body)
	}
}

func (u UdpHandler) messageHandler(src *net.UDPAddr, body []byte) {
	var msg models.Message
	err := msgpack.Unmarshal(body, &msg)
	if err != nil {
		log.Println("message unmarshal error ", err)
		return
	}
	log.Printf("msg %+v...", msg.GetPayload()[0:5])

	if msg.IsValid() {
		stored := u.MessageStorage.Set(msg)
		if u.HashStorage.Add(msg.GetHash()) && stored {
			u.Gossiper.SendMessage(msg)
		}
	} else {
		log.Println("Invalid message")
	}
}

func (u UdpHandler) welcomeHandler(src *net.UDPAddr, body []byte) {
	var wp models.WelcomePack
	err := msgpack.Unmarshal(body, &wp)
	if err != nil {
		log.Println("welcome unmarshal error ", err)
		return
	}

	u.PeerStorage.Merge(wp.PeerList)

	if !wp.Msg.IsEmpty() {
		if wp.Msg.IsValid() {
			u.MessageStorage.Set(wp.Msg)
		} else {
			log.Println("Invalid message")
		}
	}
}

func (u UdpHandler) reportHandler(src *net.UDPAddr, body []byte) {
	var msg models.Message
	err := msgpack.Unmarshal(body, &msg)
	if err != nil {
		log.Println("report unmarshal error ", err)
		return
	}
	log.Printf("%+v", msg)
}

func (u UdpHandler) monitoringHandler(src *net.UDPAddr, body []byte) {
	peer := models.Peer{IP: src.IP, Port: binary.LittleEndian.Uint16(body)}
	address := peer.ToString()

	mb, err := msgpack.Marshal(u.MessageStorage.Get())
	if err != nil {
		log.Println("monitoring marshal error ", err)
		return
	}
	payload := append(c.PrefReport, mb...)
	transport.SendPayloadToUDP(address, payload)
}

func (u UdpHandler) helloHandler(src *net.UDPAddr, body []byte) {
	peer := models.Peer{IP: src.IP, Port: binary.LittleEndian.Uint16(body)}
	address := peer.ToString()
	u.PeerStorage.Add(peer)

	wb, err := msgpack.Marshal(models.WelcomePack{PeerList: u.PeerStorage.List(), Msg: u.MessageStorage.Get()})
	if err != nil {
		log.Println("hello marshal error ", err)
		return
	}
	payload := append(c.PrefWelcome, wb...)
	transport.SendPayloadToUDP(address, payload)
}
