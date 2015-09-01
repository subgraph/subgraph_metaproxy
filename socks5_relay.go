package main

import (
	"net"
	"strconv"
	"encoding/base32"
	"fmt"

	"golang.org/x/net/proxy"
)

func relaySocks5(clientConn net.Conn, client *Client, relayPort string) (error) {
	proxyAddr := net.JoinHostPort("127.0.0.1", relayPort)
	destAddr := net.JoinHostPort(client.destAddr.String(),
		strconv.Itoa(int(client.destPort)))
	var userBytes [8]byte
	auth := proxy.Auth{
		User:     base32.StdEncoding.EncodeToString(userBytes[:]),
		Password: "",
	}
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, &auth, proxy.Direct)
	if err != nil {
		return fmt.Errorf("Error creating SOCKS5 proxy %s: %s",
			proxyAddr, err)
	}
	proxyConn, err := dialer.Dial("tcp", destAddr)
	if err != nil {
		return fmt.Errorf("Error dialing SOCKS5 proxy at %s: %s\n",
			proxyAddr, err)
	}
	go copyAndClose(proxyConn, clientConn)
	go copyAndClose(clientConn, proxyConn)
	return nil
}
