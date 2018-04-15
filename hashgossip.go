package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	c "hashgossip/consts"
	"hashgossip/messenger"
	m "hashgossip/models"
	"hashgossip/storages"
	"hashgossip/transport"

	"github.com/vmihailenco/msgpack"
	"github.com/BurntSushi/toml"
)

const (
	prefLen       = 5
)

var (
	killerFlag  = flag.Bool("killer", false, "Send shutdown signal over multicast")
	watcherFlag = flag.Bool("watcher", false, "Send monitoring signal over multicast and 10 sec receive results")

	peerStorage    storage.PeerStorage
	messageStorage storage.MessageStorage
	hashStorage    storage.HashStorage
	gossiper       messenger.Gossiper
)

func ping(address string, lAddr *net.UDPAddr) {
	payload := make([]byte, prefLen+2)
	copy(payload, c.PrefHello)
	binary.LittleEndian.PutUint16(payload[prefLen:], uint16(lAddr.Port))

	err := transport.SendPayloadToUDP(address, payload)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	var conf m.Config
	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		log.Fatal("can't read config file ", err)
	}

	flag.Parse()
	if *killerFlag {
		transport.SendPayloadToUDP(conf.MulticastAddress, c.PrefShutdown)
		os.Exit(0)
	}

	peerStorage = storage.NewPeerStorage()
	messageStorage = storage.NewMessageStorage()
	hashStorage = storage.NewHashStorage()

	gossiper = messenger.NewGossiper(peerStorage)
	go gossiper.StartLoop()

	udpListener, err := net.ListenUDP("udp4", nil)
	if err != nil {
		log.Fatal("can't start listen UDP ", err)
	}
	defer udpListener.Close()
	go transport.ServeUDP(udpListener, udpHandler)

	laddr, _ := udpListener.LocalAddr().(*net.UDPAddr)

	if *watcherFlag {
		payload := make([]byte, prefLen+2)
		copy(payload, c.PrefMonitoring)
		binary.LittleEndian.PutUint16(payload[prefLen:], uint16(laddr.Port))

		transport.SendPayloadToUDP(conf.MulticastAddress, payload)
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}

	log.SetPrefix(fmt.Sprintf("[%v]", laddr.Port))
	peerStorage.Add(m.Peer{IP: GetOutboundIP(), Port: uint16(laddr.Port)})
	go transport.ServeMulticastUDP(conf.MulticastAddress, udpHandler)

	for {
		ping(conf.MulticastAddress, laddr)
		time.Sleep(1 * time.Second)
		if !peerStorage.IsEmpty() {
			break
		}
	}

	go messenger.StartEmmitingMessages(gossiper, rand.Intn(conf.LimitMessages), conf.InvalidFrequent)

	select {}
}

func udpHandler(src *net.UDPAddr, n int, buf []byte) {
	header, body := buf[:prefLen], buf[prefLen:n]
	switch {
	case bytes.Equal(header, c.PrefMessage):
		messageHandler(src, body)
	case bytes.Equal(header, c.PrefWelcome):
		welcomeHandler(src, body)
	case bytes.Equal(header, c.PrefReport):
		reportHandler(src, body)
	case bytes.Equal(header, c.PrefMonitoring):
		monitoringHandler(src, body)
	case bytes.Equal(header, c.PrefShutdown):
		os.Exit(2)
	case bytes.Equal(header, c.PrefHello):
		helloHandler(src, body)
	}
}

func messageHandler(src *net.UDPAddr, body []byte) {
	var msg m.Message
	err := msgpack.Unmarshal(body, &msg)
	if err != nil {
		log.Println("message unmarshal error ", err)
		return
	}
	log.Printf("msg %+v...", msg.GetPayload()[0:5])

	if msg.IsValid() {
		stored := messageStorage.Set(msg)
		if hashStorage.Add(msg.GetHash()) && stored {
			gossiper.SendMessage(msg)
		}
	} else {
		log.Println("Invalid message")
	}
}

func welcomeHandler(src *net.UDPAddr, body []byte) {
	var wp m.WelcomePack
	err := msgpack.Unmarshal(body, &wp)
	if err != nil {
		log.Println("welcome unmarshal error ", err)
		return
	}

	peerStorage.Merge(wp.PeerList)

	if !wp.Msg.IsEmpty() {
		if wp.Msg.IsValid() {
			messageStorage.Set(wp.Msg)
		} else {
			log.Println("Invalid message")
		}
	}
}

func reportHandler(src *net.UDPAddr, body []byte) {
	var msg m.Message
	err := msgpack.Unmarshal(body, &msg)
	if err != nil {
		log.Println("report unmarshal error ", err)
		return
	}
	log.Printf("%+v", msg)
}

func monitoringHandler(src *net.UDPAddr, body []byte) {
	peer := m.Peer{IP: src.IP, Port: binary.LittleEndian.Uint16(body)}
	address := peer.ToString()

	mb, err := msgpack.Marshal(messageStorage.Get())
	if err != nil {
		log.Println("monitoring marshal error ", err)
		return
	}
	payload := append(c.PrefReport, mb...)
	transport.SendPayloadToUDP(address, payload)
}

func helloHandler(src *net.UDPAddr, body []byte) {
	peer := m.Peer{IP: src.IP, Port: binary.LittleEndian.Uint16(body)}
	address := peer.ToString()
	peerStorage.Add(peer)

	wb, err := msgpack.Marshal(m.WelcomePack{PeerList: peerStorage.List(), Msg: messageStorage.Get()})
	if err != nil {
		log.Println("hello marshal error ",err)
		return
	}
	payload := append(c.PrefWelcome, wb...)
	transport.SendPayloadToUDP(address, payload)
}

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp4", "8.8.8.8:80")
	if err != nil {
		log.Fatal("can't get outbound IP ", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
