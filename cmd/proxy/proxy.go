/*
Proxy is a Minetest proxy server
supporting multiple concurrent connections.

Usage:
	proxy dial:port listen:port
where dial:port is the server address
and listen:port is the address to listen on.
*/
package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/anon55555/mt"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: proxy dial:port listen:port")
		os.Exit(1)
	}

	srvaddr, err := net.ResolveUDPAddr("udp", os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	lc, err := net.ListenPacket("udp", os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	defer lc.Close()

	l := mt.Listen(lc)
	for {
		clt, err := l.Accept()
		if err != nil {
			log.Print(err)
			continue
		}

		log.Print(clt.RemoteAddr().String() + " connected")

		conn, err := net.DialUDP("udp", nil, srvaddr)
		if err != nil {
			log.Print(err)
			continue
		}
		srv := mt.Connect(conn)

		go proxy(clt, srv)
		go proxy(srv, clt)
	}
}

func proxy(src, dest mt.Peer) {
	s := fmt.Sprint(src.ID(), " (", src.RemoteAddr(), "): ")

	for {
		pkt, err := src.Recv()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				if err := src.WhyClosed(); err != nil {
					log.Print(s, "disconnected: ", err)
				} else {
					log.Print(s, "disconnected")
				}
				break
			}

			log.Print(s, err)
			continue
		}

		if _, err := dest.Send(pkt); err != nil {
			log.Print(err)
		}
	}

	dest.Close()
}
