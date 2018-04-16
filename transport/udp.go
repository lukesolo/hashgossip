package transport

import (
	"errors"
	"fmt"
	"log"
	"net"
)

const MaxDatagramSize = 8192

func SendPayloadToUDP(address string, payload []byte) error {
	if len(payload) > MaxDatagramSize {
		return errors.New(fmt.Sprintf("maxPayloadSize = %v, payload size = %v", MaxDatagramSize, len(payload)))
	}

	udpConn, err := net.Dial("udp4", address)
	if err != nil {
		return err
	}
	defer udpConn.Close()

	_, err = udpConn.Write(payload)
	return err
}

func ServeUDP(conn *net.UDPConn, handler func(*net.UDPAddr, int, []byte)) {
	buf := make([]byte, MaxDatagramSize)

	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println(src, " ", err)
			continue
		}

		// обработка входящих сообщений происходит в одной горутине,
		// что может привести к тому пир некоторе время не сможет принимать новые сообщения
		handler(src, n, buf)
	}
}

func ServeMulticastUDP(address string, handler func(*net.UDPAddr, int, []byte)) {
	multicastAddr, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		log.Fatal(err)
	}
	udpListner, err := net.ListenMulticastUDP("udp4", nil, multicastAddr)
	udpListner.SetReadBuffer(MaxDatagramSize)
	for {
		buf := make([]byte, MaxDatagramSize)
		n, src, err := udpListner.ReadFromUDP(buf)
		if err != nil {
			log.Println("ReadFromUDP failed:", err)
		}
		// здесь тоже обработка в той же горутине
		handler(src, n, buf)
	}
}
