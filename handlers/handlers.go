package handlers

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	c "hashgossip/consts"
	"hashgossip/messenger"
	"hashgossip/models"
	"hashgossip/storages"
	"hashgossip/transport"

	"github.com/vmihailenco/msgpack"
	"os"
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
		u.shutdownHandler(src, body)
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

	if u.saveMessage(msg) {
		u.Gossiper.SendMessage(msg)
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
		log.Printf("welcome msg %+v...", wp.Msg.GetPayload()[0:5])
		u.saveMessage(wp.Msg)
	}
}

func (u UdpHandler) saveMessage(msg models.Message) bool {
	if msg.IsValid() {
		stored := u.MessageStorage.Set(msg)
		if stored {
			log.Printf("new message was set %+v", msg.GetPayload()[0:5])
		}
		newHash := u.HashStorage.Add(msg.GetHash())
		return stored && newHash
	} else {
		log.Println("Invalid message")
	}
	return false
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

func (u UdpHandler) shutdownHandler(src *net.UDPAddr, body []byte) {
	log.Println("got shutdown signal")
	os.Exit(2)
}