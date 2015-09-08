package main

import (
	"encoding/json"
	"net"
	"os"
	"io"
	"log"
	"fmt"
	"reflect"
	"strconv"
	"syscall"
)

const SO_ORIGINAL_DST = 80

type Relay struct {
	DestinationPort string `json:"destinationport"`
	RelayType       RelayType `json:"relaytype"`
	RelayIP         string `json:"relayip"`
	RelayPort       string `json:"relayport"`
}

type RelayConfig struct {
	Relays []Relay
}

type Client struct {
	clientAddr net.Addr
	destAddr   net.IP
	destPort   uint16
	clientConn net.Conn
	state      bool
}

type RelayType int

const (
	SOCKS5 RelayType = iota
	HTTP_CONNECT
)

func (r RelayType) String() (string) {
	switch r {
	case SOCKS5:
		return "SOCKS5"
	case HTTP_CONNECT:
		return "HTTP_CONNECT"
	default:
		return "Unsupported relay type"
	}
}

func (r *RelayType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("relaytype should be a string, got %s", data)
	}
	relayTypes := map[string]RelayType{
		"SOCKS5": SOCKS5, 
		"HTTP_CONNECT": HTTP_CONNECT}
	rt, ok := relayTypes[s]
	if !ok {
		return fmt.Errorf("invalid RelayType %q", s)
	}
	*r = rt
	return nil
}

func ReadConfig(path string) (map[string]Relay, error) {
	var relays map[string]Relay
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	decoder := json.NewDecoder(file)
	relayConfig := RelayConfig{}
	var f os.FileInfo
	Debugf("Reading config file: %s\n", path)
	if err := decoder.Decode(&relayConfig); err != nil {
		if serr, ok := err.(*json.SyntaxError); ok {
			line, col := findLine(f, serr.Offset)
			msg := fmt.Sprintf(
				"JSON syntax error in config file %s at line %d and col %d: %s",
				path, line, col, err)
			LogWriter.Err(msg)
			os.Exit(1)
		}
	}
	if len(relayConfig.Relays) == 0 {
		msg := fmt.Sprintf("Error extracting relay info from config file: %s",
			path)
		LogWriter.Err(msg)
		os.Exit(1)
	}
	relays = make(map[string]Relay)
	for _, relay := range relayConfig.Relays {
		relays[relay.DestinationPort] = relay
	}
	return relays, nil
}

func findLine(os.FileInfo, int64) (int, int) {
	return 0, 0
}

// taken from go.et/ipv4/helper_unix.go
func sysfd(c net.Conn) (int, error) {
	cv := reflect.ValueOf(c)
	switch ce := cv.Elem(); ce.Kind() {
	case reflect.Struct:
		netfd := ce.FieldByName("conn").FieldByName("fd")
		switch fe := netfd.Elem(); fe.Kind() {
		case reflect.Struct:
			fd := fe.FieldByName("sysfd")
			return int(fd.Int()), nil
		}
	}
	return 0, nil
}

func sockaddrToIP(s []byte) net.IP {
	return net.IP(s)
}

func acceptClient(conn *net.TCPConn, relays map[string]Relay) {
	client := Client{}
	destAddr, destPort, newConn, err := getDestAddr(conn)
	if err != nil {
		msg := fmt.Sprintf("Error getting original destAddr in acceptClient: %s",
			err)
		LogWriter.Err(msg)
	}
	client.clientAddr = conn.RemoteAddr()
	client.destAddr = destAddr
	client.destPort = destPort
	client.clientConn = conn
	Debugf("Client clientAddr: %s\n", client.clientAddr)
	Debugf("Client destAddr: %s\n", client.destAddr)
	Debugf("Client destPort: %d\n", client.destPort)
	handleProxyConnection(newConn, &client, relays)
}

func getDestAddr(clientConn *net.TCPConn) (net.IP, uint16, *net.TCPConn, error) {
	// http://lkml.iu.edu/hypermail/linux/kernel/9910.0/0221.html
	// http://comments.gmane.org/gmane.comp.lang.go.general/108326
	var destAddr net.IP
	var destPort uint16
	var newTCPConn *net.TCPConn
	fd, err := sysfd(clientConn)
	if err != nil {
		msg := fmt.Sprintf("Error get net.Conn fd in getDestAddr: %s",
			err)
		LogWriter.Err(msg)
	}
	addr, err := syscall.GetsockoptIPv6Mreq(fd,
		syscall.IPPROTO_IP, SO_ORIGINAL_DST)
	if err != nil {
		msg := fmt.Sprintf("Err in syscall.Getsockopt in getDestAddr: %s",
			err)
		LogWriter.Err(msg)
		return destAddr, destPort, newTCPConn, err
	}
	newTCPConn = clientConn
	destAddr = sockaddrToIP(addr.Multiaddr[4:8])
	destPort = uint16(addr.Multiaddr[2])<<8 + uint16(addr.Multiaddr[3])
	return destAddr, destPort, newTCPConn, nil
}

func ProxyRelay(port string, relays map[string]Relay) (error) {
	tcpAddr := net.TCPAddr{}
	tcpAddr.IP = net.ParseIP("127.0.0.1")
	tcpAddr.Port, _ = strconv.Atoi(port)
	ln, err := net.ListenTCP("tcp", &tcpAddr)
	if err != nil {
		msg := fmt.Sprintf("Proxy listener error on port %s: %s",
			port, err)
		LogWriter.Err(msg)
		return fmt.Errorf(msg)
	}
	msg := fmt.Sprintf("Started proxy listener on port: %s",
		port)
	LogWriter.Info(msg)
	defer ln.Close()
	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			msg := fmt.Sprintf("Error accepting TCP connection on port %s: %s",
				port, err)
			LogWriter.Err(msg)
			continue
		}
		conn.SetKeepAlive(true)
		if err != nil {
			msg := fmt.Sprintf("Error setting keep-alive on connection: %s",
				err)
			LogWriter.Err(msg)
		}
		go acceptClient(conn, relays)
	}
	return nil
}

func handleProxyConnection(conn net.Conn, client *Client, relays map[string]Relay) (error) {
	dpString := strconv.Itoa(int(client.destPort))
	var relay Relay
	// Look for relay configuration for destination port
	if _, ok := relays[dpString]; ok {
		relay, _ = relays[dpString]
	} else {
		// No configuration for destination found, check if a wildcard configuration exists
		relay, ok = relays["*"]
		if ok {
			msg := fmt.Sprintf(
				"No specific policy for client destination port, using wildcard relay for port: %s\n",
				dpString)
			LogWriter.Info(msg)
		} else {
			// No wildcard policy found, connection will be dropped
			msg := fmt.Sprintf("No policy for client destination port %s, dropping connection",
				dpString)
			LogWriter.Info(msg)
			return nil
		}
	}
	switch relay.RelayType {
	case HTTP_CONNECT:
		LogWriter.Info("Opening HTTP/S CONNECT relay.")
		err := relayHttp(conn, client, relay)
		if err != nil {
			LogWriter.Err(err.Error())
		}
	case SOCKS5:
		LogWriter.Info("Opening SOCKS5 relay.")
		err := relaySocks5(conn, client, relay)
		if err != nil {
			LogWriter.Err(err.Error())
		}
	default:
		return fmt.Errorf("Relay type not found: %s\n", relay.RelayType)
	}
	return nil
}

func copyAndClose(w io.WriteCloser, r io.Reader) {
	written, err := io.Copy(w, r)
	if err != nil {
		Debugf("Error copying data: %s\n", err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			msg := fmt.Sprintf("Error closing destination in copyAndClose: %s",
				err)
			LogWriter.Info(msg)
			return
		}
	}()

	Debugf("Bytes written: %d\n", written)
	return
}
