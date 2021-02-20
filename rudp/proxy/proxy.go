/*
Proxy is a Minetest RUDP proxy server
supporting multiple concurrent connections.

Usage:
	proxy dial:port listen:port
where dial:port is the server address
and listen:port is the address to listen on.
*/
package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"mt/rudp"
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

	l := rudp.Listen(lc)
	for {
		clt, err := l.Accept()
		if err != nil {
			log.Print(err)
			continue
		}

		log.Print(clt.Addr(), " connected")

		conn, err := net.DialUDP("udp", nil, srvaddr)
		if err != nil {
			log.Print(err)
			continue
		}
		srv := rudp.Connect(conn, conn.RemoteAddr())

		go proxy(clt, srv)
		go proxy(srv, clt)
	}
}

func proxy(src, dest *rudp.Peer) {
	for {
		pkt, err := src.Recv()
		if err != nil {
			if err == net.ErrClosed {
				msg := src.Addr().String() + " disconnected"
				if src.TimedOut() {
					msg += " (timed out)"
				}
				log.Print(msg)

				break
			}

			log.Print(err)
			continue
		}

		if _, err := dest.Send(pkt); err != nil {
			log.Print(err)
		}
	}

	dest.SendDisco(0, true)
	dest.Close()
}
