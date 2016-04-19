package tcp

//Explanation: http://synflood.at/tmp/golang-slides/mrmcd2012.html
//Source: https://github.com/akrennmair/telnet-chat/blob/master/03_chat/chat.go
//Test fra shell  I: echo -n "Random string" | nc localhost 6000
//Test fra shell II: nc localhost 6000

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

const (
	tcpPort = ":6000"
)


type Client struct {
	id   string
	conn net.Conn
	ch   chan string
}

type RawMessage struct{
	data []byte
	ip string
}

func (c Client) RecieveFrom(ch chan<- string) {
	bufc := bufio.NewReader(c.conn)
	for {
		line, err := bufc.ReadString('\n')
		if err != nil {
			log.Println(err)
			break
		}
		ch <- fmt.Sprintf("%s: %s", c.id, line) //Denne blokkerer frem til en ny melding er mottatt
	}
}

func (c Client) SendTo(ch <-chan string) {
	for msg := range ch { //Denne loopen trigges hver gang en ny melding kommer inn
		_, err := io.WriteString(c.conn, msg)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func handleMessages(sendchan <-chan string, addchan <-chan Client, rmchan <-chan Client, localAddress string, tcpConnected chan string, tcpConnectionFailure chan string) {
	clients := make(map[net.Conn]chan<- string)

	for {
		select {
		case msg := <-sendchan:
			//Broadcast on TCP
			for _, ch := range clients {
				go func(mch chan<- string) {
					mch <- "\033[1;33;30m" + msg + "\033[m\r\n"
				}(ch)
			}
		case client := <-addchan:
			clients[client.conn] = client.ch
			tcpConnected <- client.id
		case client := <-rmchan:
			log.Printf("Disconnected: %s\n", client.id)
			delete(clients, client.conn)
			tcpConnectionFailure <- client.id
		case <-time.Tick(10 * time.Second):
			//Send heartbeat on TCP
			for _, ch := range clients {
				go func(mch chan<- string) {
					mch <- "\033[1;33;30m" + "TCP hearbeat from " + localAddress + "\033[m\r\n"
				}(ch)
			}
		}
	}
}

func handleConnection(connection net.Conn, recvchan chan<- string, addchan chan<- Client, rmchan chan<- Client) {

	defer connection.Close()

	client := Client{
		id:   connection.RemoteAddr().String(),
		conn: connection,
		ch:   make(chan string),
	}

	addchan <- client

	defer func() {
		log.Printf("Connection from %v closed.\n", connection.RemoteAddr())
		rmchan <- client
	}()
	//msgchan <- fmt.Sprintf("New elevator %s has connected.\n", client.id)

	// I/O
	go client.RecieveFrom(recvchan)
	client.SendTo(client.ch)
}

func listen(recvchan chan<- string, addchan chan<- Client, rmchan chan<- Client) {

	listener, err := net.Listen("tcp", tcpPort)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening for TCP connections on %v", listener.Addr())

	for {
		connection, err := listener.Accept()

		if err != nil {
			log.Println(err)
			continue
		}
		log.Printf("Handling incoming connection from %v", connection.RemoteAddr())
		go handleConnection(connection, recvchan, addchan, rmchan)
	}
}

func dial(remoteIp string, recvchan chan<- string, addchan chan<- Client, rmchan chan<- Client) {
	connection, err := net.Dial("tcp", remoteIp+tcpPort)

	for {
		if err != nil {
			log.Printf("TCP dial to %s failed", remoteIp+tcpPort)
			time.Sleep(500 * time.Millisecond)
			connection, err = net.Dial("tcp", remoteIp+tcpPort) //Annen måte å gjøre dette på?
		} else {
			log.Println("Handling dialed connection to ",remoteIp)
			go handleConnection(connection, recvchan, addchan, rmchan)
			break
		}
	}
}

func Init(tcpSendMsg, tcpRecvMsg, tcpConnected, tcpConnectionFailure, tcpDial chan string, localAddress string) {

	addchan := make(chan Client)
	rmchan := make(chan Client)

	go handleMessages(tcpSendMsg, addchan, rmchan, localAddress, tcpConnected, tcpConnectionFailure)
	go listen(tcpRecvMsg, addchan, rmchan)

	for {
		remoteIp := <-tcpDial
		log.Println("Dialing ",remoteIp)
		go dial(remoteIp, tcpRecvMsg, addchan, rmchan)
	}
}
