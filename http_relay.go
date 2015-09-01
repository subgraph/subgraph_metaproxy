package main

import (
        "net"
        "net/url"
        "net/http"
        "strconv"
        "bufio"
        "fmt"
)


func relayHttp(clientConn net.Conn, client *Client, relayPort string) (error) {
        // TODO: Proxy IP Should not be hard-coded, needs to come from config
        proxyAddr := net.JoinHostPort("127.0.0.1", relayPort)
        destAddr := net.JoinHostPort(client.destAddr.String(),
                strconv.Itoa(int(client.destPort)))
        destUrl, err := url.Parse(destAddr)
        if err != nil {
                return fmt.Errorf("Invalid destAddr %s: %s", destAddr, err)
        }
        proxyConn, err := net.Dial("tcp", proxyAddr)
        if err != nil {
                return fmt.Errorf("Error dialing HTTP proxy %s: %s", proxyAddr, err)
        }
       	connectReq := &http.Request{
                Method: "CONNECT",
                URL: destUrl,
                Host: proxyAddr,
                Header: make(http.Header),
        }
        connectReq.Write(proxyConn)
        br := bufio.NewReader(proxyConn)
        resp, err := http.ReadResponse(br, connectReq)
        if err != nil {
                proxyConn.Close()
                return err
        }
        if resp.StatusCode != 200 {
                proxyConn.Close()
                return nil
        }
        go copyAndClose(proxyConn, clientConn)
        go copyAndClose(clientConn, proxyConn)
        return nil
}


