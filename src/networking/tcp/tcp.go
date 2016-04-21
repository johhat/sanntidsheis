package tcp

//Explanation: http://synflood.at/tmp/golang-slides/mrmcd2012.html
//Source: https://github.com/akrennmair/telnet-chat/blob/master/03_chat/chat.go
//Test fra shell  I: echo -n "Random string" | nc localhost 6000
//Test fra shell II: nc localhost 6000

//TODO: Legg inn feilhåndtering ved feil i lesing fra tilkobling
//TODO: Vurder navnsettingen. ip vs. id

import (
	"bufio"
	"bytes"
	"log"
	"net"
	"strings"
	"time"
)

const (
	tcpPort      = ":6000"
	writeTimeout = 10 * time.Second
)

type Client struct {
	id   string
	conn net.Conn
	ch   chan []byte
}

type RawMessage struct {
	Data []byte
	Ip   string
}

func (c Client) RecieveFrom(ch chan<- RawMessage, closeChan chan<- bool) {

	reader := bufio.NewReader(c.conn)

	for {
		c.conn.SetReadDeadline(time.Now().Add(writeTimeout))

		bytes, err := reader.ReadBytes('\n')

		if err != nil {
			log.Println("TCP recv error. Connection: ", c.id, " error:", err)
			closeChan <- true
			return
		}
		ch <- RawMessage{Data: bytes, Ip: c.id}
	}
}

func (c Client) SendTo(ch <-chan []byte, closeChan chan<- bool) {

	var b bytes.Buffer

	for msg := range ch {

		b.Write(msg)
		b.WriteRune('\n')

		_, err := c.conn.Write(b.Bytes())

		b.Reset()

		if err != nil {
			log.Println("TCP send error. Connection: ", c.id, " error:", err)
			closeChan <- true
			return
		}
	}
}

func handleMessages(sendMsg <-chan RawMessage, broadcastMsg <-chan []byte, addchan <-chan Client, rmchan <-chan Client, localAddress string, tcpConnected chan string, tcpConnectionFailure chan string) {

	clients := make(map[net.Conn]chan<- []byte)

	for {
		select {
		case rawMsg := <-sendMsg:
			sendToId(rawMsg.Ip, clients, rawMsg.Data)
		case msg := <-broadcastMsg:
			broadcast(clients, msg)
		case client := <-addchan:
			clients[client.conn] = client.ch
			tcpConnected <- client.id
		case client := <-rmchan:
			delete(clients, client.conn)
			tcpConnectionFailure <- client.id
		}
	}
}

func sendToId(id string, clients map[net.Conn]chan<- []byte, message []byte) {
	//TODO: Må testes
	for connection, channel := range clients {
		if getRemoteIp(connection) == id {
			channel <- message
			break
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
		id:   getRemoteIp(connection),
		conn: connection,
		ch:   make(chan []byte),
	}

	addchan <- client

	defer func() {
		log.Printf("Connection from %v closed.\n", connection.RemoteAddr())
		rmchan <- client
	}()

	// I/O
	closeChan := make(chan bool)

	go client.RecieveFrom(recvchan, closeChan)
	go client.SendTo(client.ch, closeChan)

	<-closeChan
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

func getRemoteIp(connection net.Conn) string {
	return strings.Split(connection.RemoteAddr().String(), ":")[0]
}

func Init(tcpSendMsg <-chan RawMessage, tcpBroadcastMsg <-chan []byte, tcpRecvMsg chan RawMessage, tcpConnected, tcpConnectionFailure, tcpDial chan string, localAddress string) {

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
