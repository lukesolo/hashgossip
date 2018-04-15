package main

import (
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

	"github.com/BurntSushi/toml"
	"hashgossip/handlers"
)

var (
	killerFlag  = flag.Bool("killer", false, "Send shutdown signal over multicast")
	watcherFlag = flag.Bool("watcher", false, "Send monitoring signal over multicast and 10 sec receive results")
)

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

	peerStorage := storage.NewPeerStorage()
	messageStorage := storage.NewMessageStorage()
	hashStorage := storage.NewHashStorage()
	gossiper := messenger.NewGossiper(peerStorage)

	udpHandler := handlers.UdpHandler{
		PeerStorage:    peerStorage,
		MessageStorage: messageStorage,
		HashStorage:    hashStorage,
		Gossiper:       gossiper,
	}
	go gossiper.StartLoop()

	udpListener, err := net.ListenUDP("udp4", nil)
	if err != nil {
		log.Fatal("can't start listen UDP ", err)
	}
	defer udpListener.Close()
	go transport.ServeUDP(udpListener, udpHandler.Handler)

	laddr, _ := udpListener.LocalAddr().(*net.UDPAddr)

	if *watcherFlag {
		payload := make([]byte, c.PrefLen+2)
		copy(payload, c.PrefMonitoring)
		binary.LittleEndian.PutUint16(payload[c.PrefLen:], uint16(laddr.Port))

		transport.SendPayloadToUDP(conf.MulticastAddress, payload)
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}

	log.SetPrefix(fmt.Sprintf("[%v]", laddr.Port))
	peerStorage.Add(m.Peer{IP: getOutboundIP(), Port: uint16(laddr.Port)})
	go transport.ServeMulticastUDP(conf.MulticastAddress, udpHandler.Handler)

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

func ping(address string, lAddr *net.UDPAddr) {
	payload := make([]byte, c.PrefLen+2)
	copy(payload, c.PrefHello)
	binary.LittleEndian.PutUint16(payload[c.PrefLen:], uint16(lAddr.Port))

	err := transport.SendPayloadToUDP(address, payload)
	if err != nil {
		log.Fatal(err)
	}
}

// Get preferred outbound ip of this machine
func getOutboundIP() net.IP {
	conn, err := net.Dial("udp4", "8.8.8.8:80")
	if err != nil {
		log.Fatal("can't get outbound IP ", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
