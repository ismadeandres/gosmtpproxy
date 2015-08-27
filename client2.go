package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"runtime"
	"strconv"
	"time"
)

var config = map[string]string{
	"listen": "localhost:10025",
	"plug":   "localhost:10026",
}

func configure() {

	var listen, plug string

	flag.StringVar(&listen, "listen", "localhost:10025", "listenAddress:port")
	flag.StringVar(&plug, "plug", "localhost:10026", "plugAddress:port")

	flag.Parse()

	config["listen"] = listen
	config["plug"] = plug
}

type Client struct {
	read_buffer string
	response    string
	address     string
	hash        string
	time        int64
	conn        net.Conn
	bufin       *bufio.Reader
	bufout      *bufio.Writer
	kill_time   int64
	errors      int
	clientId    int64
}

type Target struct {
	read_buffer string
	response    string
	address     string
	hash        string
	time        int64
	conn        net.Conn
	bufin       *bufio.Reader
	bufout      *bufio.Writer
	errors      int
}

func main() {

	configure()

	// Start listening for SMTP connections
	listener, err := net.Listen("tcp", config["listen"])
	if err != nil {
		log.Printf("Cannot listen on port, %v", err)

	} else {
		log.Printf("Listening on tcp %s", config["listen"])
	}

	target, err := net.Dial("tcp", config["plug"])
	if err != nil {
		log.Printf("Cannot connet to targe, %v", err)

	} else {
		log.Printf("Connected to tcp %s", config["plug"])
	}
	defer closeTarget(target)

	t := Target{
		conn:    target,
		address: target.RemoteAddr().String(),
		time:    time.Now().Unix(),
		bufin:   bufio.NewReader(target),
		bufout:  bufio.NewWriter(target),
	}

	var clientId int64
	clientId = 1
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(1, fmt.Sprintf("Accept error: %s", err))
			continue
		}
		log.Printf(" There are now " + strconv.Itoa(runtime.NumGoroutine()) + " serving goroutines")

		go handleClient(&Client{
			conn:     conn,
			address:  conn.RemoteAddr().String(),
			time:     time.Now().Unix(),
			bufin:    bufio.NewReader(conn),
			bufout:   bufio.NewWriter(conn),
			clientId: clientId,
		}, &t)

	}

}

func handleClient(client *Client, target *Target) {
	defer closeClient(client)
	for {
		//target.bufin.ReadString('\n')
		line, _, _ := target.bufin.ReadLine()
		client.bufout.WriteString(string(line))
	}
	readSmtp(client)

	log.Println(client.time)

}

func readSmtp(client *Client) {

}

func closeTarget(t net.Conn) {
	t.Close()
}

func closeClient(client *Client) {
	client.conn.Close()

}
