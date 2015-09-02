package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var config = map[string]string{
	"listen": "localhost:10025",
	"plug":   "localhost:10026",
}

var timeout time.Duration = 100
var maxSize = 131072

func configure() {

	var listen, plug string

	flag.StringVar(&listen, "listen", "localhost:10025", "listenAddress:port")
	flag.StringVar(&plug, "plug", "localhost:10026", "plugAddress:port")

	flag.Parse()

	config["listen"] = listen
	config["plug"] = plug

}

/*
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
}*/

type Speaker struct {
	state      int
	helo       string
	mailFrom   string
	rcptTo     string
	readBuffer string
	response   string
	address    string
	data       string
	hash       string
	time       int64
	conn       net.Conn
	bufin      *bufio.Reader
	bufout     *bufio.Writer
	killTime   int64
	errors     int
	id         int64
	role       string
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

	var clientID int64
	clientID = 1
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(1, fmt.Sprintf("Accept error: %s", err))
			continue
		}
		log.Printf(" There are now " + strconv.Itoa(runtime.NumGoroutine()) + " serving goroutines")

		target, err := net.Dial("tcp", config["plug"])
		if err != nil {
			log.Printf("Cannot connet to target, %v", err)

		} else {
			log.Printf("Connected to tcp %s", config["plug"])
		}
		defer closeTarget(target)

		t := Speaker{
			conn:    target,
			address: target.RemoteAddr().String(),
			time:    time.Now().Unix(),
			bufin:   bufio.NewReader(target),
			bufout:  bufio.NewWriter(target),
			role:    "server",
		}

		go handleClient(&Speaker{
			conn:    conn,
			address: conn.RemoteAddr().String(),
			time:    time.Now().Unix(),
			bufin:   bufio.NewReader(conn),
			bufout:  bufio.NewWriter(conn),
			id:      clientID,
			role:    "client",
		}, &t)

	}

}

func handleClient(client *Speaker, target *Speaker) {
	defer closeSpeaker(client)

	for i := 0; i < 100; i++ {
		switch client.state {
		case 0:

			line, _ := readSMTP(target)
			responseAdd(client, line)
			responseWrite(client)

			client.state = 1

		case 1:

			input, err := readSMTP(client)
			responseAdd(target, input)
			responseWrite(target)

			if err != nil {
				log.Println(1, fmt.Sprintf("Read error: %v", err))
				if err == io.EOF {
					// client closed the connection already
					return
				}
				if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
					// too slow, timeout
					return
				}
				break
			}
			input = strings.Trim(input, " \n")
			cmd := strings.ToUpper(input)

			switch {

			case strings.Index(cmd, "HELO") == 0:

				if len(input) > 5 {
					client.helo = input[5:]
				}
				//responseAdd(client, "250 pormisweb.com Hello ")
			case strings.Index(cmd, "EHLO") == 0:
				log.Println("Atencion que va un " + cmd)
				if len(input) > 5 {
					client.helo = input[5:]
				}
				/*
					responseAdd(client, "250-"+"pormiswebs.com"+" Hello "+client.helo+"["+client.address+"]"+"\r\n"+"250-SIZE "+"10240000"+"\r\n"+"250-HELP\n")
					responseWrite(client)
					break*/

			case strings.Index(cmd, "MAIL FROM:") == 0:
				if len(input) > 10 {
					client.mailFrom = input[10:]
				}
				//responseAdd(client, "250 Ok")

			case strings.Index(cmd, "RCPT TO:") == 0:
				if len(input) > 8 {
					client.rcptTo = input[8:]
				}
				//responseAdd(client, "250 Accepted")
			case strings.Index(cmd, "NOOP") == 0:
				//responseAdd(client, "250 OK")
			case strings.Index(cmd, "RSET") == 0:
				client.mailFrom = ""
				client.rcptTo = ""
				//responseAdd(client, "250 OK")
			case strings.Index(cmd, "DATA") == 0:
				//responseAdd(client, "354 Enter message, ending with \".\" on a line by itself")
				client.state = 2
				log.Println("pasamos a Estado 2")

			case strings.Index(cmd, "QUIT") == 0:
				//responseAdd(client, "221 Bye")
				killClient(client)
			default:
				//responseAdd(client, fmt.Sprintf("500 unrecognized command"))
				client.errors++
				if client.errors > 3 {
					//responseAdd(client, fmt.Sprintf("500 Too many unrecognized commands"))
					killClient(client)
				}
			}

			var o = ""
			var sal = ""
			if strings.Index(cmd, "EHLO") == 0 {
				var er error

				for er == nil {
					sal, er = target.bufin.ReadString('\n')
					o = o + sal

					if strings.HasPrefix(sal, "250 ") {
						break
					}
				}

			} else {
				o, _ = readSMTP(target)
			}
			output := o

			responseAdd(client, output)
			responseWrite(client)

		case 2:
			var err error
			client.data, err = readSMTP(client)
			log.Println(client.data)
			if err == nil {
				// to do: timeout when adding to SaveMailChan
				// place on the channel so that one of the save mail workers can pick it up
				//SaveMailChan <- client
				// wait for the save to complete
				//status := <-client.savedNotify
				log.Println("client.data: " + client.data)
				responseAdd(target, client.data)
				responseWrite(target)

				output, _ := readSMTP(target)
				responseAdd(client, output)
				responseWrite(client)

			} else {
				log.Printf("DATA read error: %v", err)
			}
			client.state = 1
		default:
			client.errors++
			if client.errors > 3 {

				killClient(client)
			}
		}

		if client.killTime > 1 {
			return
		}
	}
}

func responseAdd(c *Speaker, line string) {
	c.response += line //+ "\r\n"
}

func killClient(client *Speaker) {
	client.killTime = time.Now().Unix()
}

func responseWrite(client *Speaker) (err error) {
	var size int
	client.conn.SetDeadline(time.Now().Add(timeout * time.Second))
	size, err = client.bufout.WriteString(client.response)
	client.bufout.Flush()
	client.response = client.response[size:]
	return err
}

func readSMTP(client *Speaker) (input string, err error) {
	var reply string
	// Command state terminator by default
	suffix := "\n"
	if client.state == 2 {
		// DATA state
		suffix = "\r\n.\r\n"
	}
	for err == nil {
		client.conn.SetDeadline(time.Now().Add(timeout * time.Second))
		reply, err = client.bufin.ReadString('\n')
		if reply != "" {
			input = input + reply
			if len(input) > maxSize {
				err = errors.New("Maximum DATA size exceeded (" + strconv.Itoa(maxSize) + ")")
				return input, err
			} /*
				if client.state == 2 {
					// Extract the subject while we are at it.
					scanSubject(client, reply)
				}*/
		}
		if err != nil {
			break
		}
		if strings.HasSuffix(input, suffix) {
			break
		}
		log.Println(input)
	}
	if client.role == "client" {
		log.Printf(" C > %v", input)
	} else {
		log.Printf(" S > %v", input)
	}
	return input, err

}

func closeTarget(t net.Conn) {
	t.Close()
}

func closeSpeaker(spk *Speaker) {
	spk.conn.Close()

}
