package main

import (
	"flag"
	"fmt"
	"log"
	"log/syslog"
	"os"
	"runtime"
	"strconv"
)

var portFlag int
var configFile string
var testConfig bool
var debug bool
var LogWriter *syslog.Writer
var SyslogError error


func init() {
	LogWriter, SyslogError = syslog.New(syslog.LOG_DEBUG, "subgraph-metaproxy")
	if SyslogError != nil {
		log.Fatal(SyslogError)
	}
	flag.IntVar(&portFlag, "p", 8675, "metaproxy port")
	flag.StringVar(&configFile, "c", "", "metaproxy config file")
	flag.BoolVar(&testConfig, "t", false, "test config file and exit")
	flag.BoolVar(&debug, "d", false, "enable debugging mode")
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s: \n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func Debugf(format string, args ...interface{}) {
	if debug {
		msg := fmt.Sprintf("[DEBUG] "+format, args)
		LogWriter.Debug(msg)
	}
}

func main() {
	flag.Parse()
	if configFile == "" {
		usage()
	}
	runtime.GOMAXPROCS(2)
	port := strconv.Itoa(portFlag)
	relayConfig, err := ReadConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}
	if testConfig {
		fmt.Printf("Relay Config: %#v\n", relayConfig)
		os.Exit(0)
	}
	err = ProxyRelay(port, relayConfig)
	if err != nil {
		log.Fatalf("Error %s, could not create ProxyRelay on port %s",
			err, port)
	}
}
