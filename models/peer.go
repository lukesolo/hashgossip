package models

import (
	"fmt"
	"net"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func (p Peer) ToString() string {
	return fmt.Sprintf("%v:%v", p.IP, p.Port)
}
