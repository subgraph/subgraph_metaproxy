package main

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"net"
	"strconv"
	"strings"

	"golang.org/x/net/proxy"
)

func relaySocks5(clientConn net.Conn, client *Client, relay Relay) error {
	userBytes := make([]byte, 8)
	var proxyAddr string
	var proxyType string
	_, err := rand.Read(userBytes)

	if err != nil {
		return err
	}
	auth := proxy.Auth{
		User:     base32.StdEncoding.EncodeToString(userBytes[:]),
		Password: "password",
	}
	if strings.HasPrefix(relay.RelayIP, "unix:") {
		proxyAddr = strings.Split(relay.RelayIP, ":")[1]
		proxyType = "unix"
	} else {
		proxyAddr = net.JoinHostPort(relay.RelayIP, relay.RelayPort)
		proxyType = "tcp"
	}
	dialer, err := proxy.SOCKS5(proxyType, proxyAddr, &auth, proxy.Direct)
	if err != nil {
		return fmt.Errorf("Error creating SOCKS5 proxy %s: %s",
			proxyAddr, err)
	}
	destAddr := net.JoinHostPort(client.destAddr.String(),
		strconv.Itoa(int(client.destPort)))
	proxyConn, err := dialer.Dial("tcp", destAddr)
	if err != nil {
		return fmt.Errorf("Error dialing SOCKS5 proxy at %s: %s\n",
			proxyAddr, err)
	}
	go copyAndClose(proxyConn, clientConn)
	go copyAndClose(clientConn, proxyConn)
	return nil
}
