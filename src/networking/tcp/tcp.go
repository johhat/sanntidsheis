package tcp

//Explanation: http://synflood.at/tmp/golang-slides/mrmcd2012.html
//Source: https://github.com/akrennmair/telnet-chat/blob/master/03_chat/chat.go
//Test fra shell  I: echo -n "Random string" | nc localhost 6000
//Test fra shell II: nc localhost 6000

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"time"
)

const (
	tcpPort           = ":6000"
	heartBeatInterval = 5 * time.Second
)

type Client struct {
	id   string
	conn net.Conn
	ch   chan []byte
}

type RawMessage struct {
	data []byte
	ip   string
}

func (msg RawMessage) String() string {
	return fmt.Sprintf("Message from %s: %s", msg.ip, string(msg.data))
}

func (c Client) RecieveFrom(ch chan<- RawMessage) {

	reader := bufio.NewReader(c.conn)

	for {
		bytes, err := reader.ReadBytes('\n')
		if err != nil {
			log.Println("Connection ", c.id, " error:", err)
			break
		}
		ch <- RawMessage{data: bytes, ip: c.id}
	}
}

func (c Client) SendTo(ch <-chan []byte) {

	var b bytes.Buffer

	for msg := range ch {

		b.Write(msg)
		b.Write([]byte("\n"))

		_, err := c.conn.Write(b.Bytes())

		if err != nil {
			log.Println(err)
			return
		}
	}
}

func handleMessages(sendMsg, broadcastMsg <-chan []byte, addchan <-chan Client, rmchan <-chan Client, localAddress string, tcpConnected chan string, tcpConnectionFailure chan string) {

	clients := make(map[net.Conn]chan<- []byte)

	tick := time.Tick(heartBeatInterval)

	for {
		select {
		case msg := <-sendMsg:
			//TODO: Implement send to one computer
			log.Print("Send to one computer placeholder. Msg: ", string(msg))
		case msg := <-broadcastMsg:
			broadcast(clients, msg)
		case client := <-addchan:
			clients[client.conn] = client.ch
			tcpConnected <- client.id
		case client := <-rmchan:
			log.Printf("Disconnected: %s\n", client.id)
			delete(clients, client.conn)
			tcpConnectionFailure <- client.id
		case <-tick:
			broadcast(clients, []byte("TCP heartbeat from "+localAddress))
		}
	}
}

func broadcast(clients map[net.Conn]chan<- []byte, message []byte) {
	for _, channel := range clients {
		go func(messageChannel chan<- []byte) {
			messageChannel <- message
		}(channel)
	}
}

func handleConnection(connection net.Conn, recvchan chan<- RawMessage, addchan chan<- Client, rmchan chan<- Client) {

	defer connection.Close()

	client := Client{
		id:   connection.RemoteAddr().String(),
		conn: connection,
		ch:   make(chan []byte),
	}

	addchan <- client

	defer func() {
		log.Printf("Connection from %v closed.\n", connection.RemoteAddr())
		rmchan <- client
	}()

	// I/O
	go client.RecieveFrom(recvchan)
	client.SendTo(client.ch)
}

func listen(recvchan chan<- RawMessage, addchan chan<- Client, rmchan chan<- Client) {

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

func dial(remoteIp string, recvchan chan<- RawMessage, addchan chan<- Client, rmchan chan<- Client) {
	connection, err := net.Dial("tcp", remoteIp+tcpPort)

	for {
		if err != nil {
			log.Printf("TCP dial to %s failed", remoteIp+tcpPort)
			time.Sleep(500 * time.Millisecond)
			connection, err = net.Dial("tcp", remoteIp+tcpPort) //TODO: Annen måte å gjøre dette på?
		} else {
			log.Println("Handling dialed connection to ", remoteIp)
			go handleConnection(connection, recvchan, addchan, rmchan)
			break
		}
	}
}

func Init(tcpSendMsg, tcpBroadcastMsg chan []byte, tcpRecvMsg chan RawMessage, tcpConnected, tcpConnectionFailure, tcpDial chan string, localAddress string) {

	addchan := make(chan Client)
	rmchan := make(chan Client)

	go handleMessages(tcpSendMsg, tcpBroadcastMsg, addchan, rmchan, localAddress, tcpConnected, tcpConnectionFailure)
	go listen(tcpRecvMsg, addchan, rmchan)

	for {
		remoteIp := <-tcpDial
		log.Println("Dialing ", remoteIp)
		go dial(remoteIp, tcpRecvMsg, addchan, rmchan)
	}
}
