package main

import (
	"log"
	"net"
	"os"
	"time"

	"github.com/miekg/dns"
	"github.com/oosawy/multicast"
)

var (
	zeroAddrUDP4 = &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 0,
	}
	mdnsAddrUDP4 = &net.UDPAddr{
		IP:   net.IPv4(224, 0, 0, 251),
		Port: 5353,
	}
	mdnsMulticastTTL = 255
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <name to query>", os.Args[0])
	}
	name := dns.Fqdn(os.Args[1])

	conn, err := multicast.ListenMulticastUDPIfaces("udp4", nil, zeroAddrUDP4)
	if err != nil {
		log.Fatal("Failed to create connection:", err)
	}
	defer conn.Close()

	conn.JoinMulticastGroup(nil, mdnsAddrUDP4)

	conn.SetMulticastTTL(mdnsMulticastTTL)
	conn.SetMulticastLoopback(true)
	conn.ReuseAddrPort()

	log.Printf("Listening on %s", conn.LocalAddr())

	sendQuery(conn, mdnsAddrUDP4, name)

	buf := make([]byte, 65536)
	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			log.Printf("Error reading: %v", err)
			continue
		}

		msg := new(dns.Msg)
		if err := msg.Unpack(buf[:n]); err != nil {
			log.Printf("Error unpacking DNS message: %v", err)
			continue
		}

		for _, q := range msg.Answer {
			log.Printf("Received DNS answer:\n\t%s", q.String())
		}

		os.Exit(0)
	}
}

func sendQuery(conn *multicast.UDPConn, addr *net.UDPAddr, name string) {
	msg := new(dns.Msg)
	msg.SetQuestion(name, dns.TypeA)
	msg.RecursionDesired = false

	buf, err := msg.Pack()
	if err != nil {
		log.Printf("Error packing DNS message: %v", err)
		return
	}

	time.Sleep(100 * time.Millisecond)

	_, err = conn.WriteTo(buf, addr)
	if err != nil {
		log.Printf("Error sending query: %v", err)
	} else {
		log.Println("Query sent")
	}
}
