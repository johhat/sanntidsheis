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
	ch   chan []byte
}

type RawMessage struct {
	data []byte
	ip   string
}

func (msg RawMessage) String() string {
	return fmt.Sprintf("Message from %s: \n %s", msg.ip, string(msg.data))
}

func (c Client) RecieveFrom(ch chan<- RawMessage) {
	bufc := bufio.NewReader(c.conn)
	for {
		line, err := bufc.ReadString('\n') //TODO: Erstatt med byte-lesing
		if err != nil {
			log.Println("Connection ", c.id, " error:", err)
			break
		}
		ch <- RawMessage{data: []byte(line), ip: c.id}
	}
}

func (c Client) SendTo(ch <-chan []byte) {

	for msg := range ch {
		_, err := io.WriteString(c.conn, string(msg)+"\n") //TODO: Erstatt med byte-skriving
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func handleMessages(sendMsg, broadcastMsg <-chan []byte, addchan <-chan Client, rmchan <-chan Client, localAddress string, tcpConnected chan string, tcpConnectionFailure chan string) {
	clients := make(map[net.Conn]chan<- []byte)

	for {
		select {
		case msg := <-sendMsg:
			//TODO: Implement send to one computer
			log.Print("Send to one computer placeholder. Msg: ", string(msg))
		case msg := <-broadcastMsg:
			//Broadcast on TCP
			for _, channel := range clients {
				go func(messageChannel chan<- []byte) {
					messageChannel <- msg
				}(channel)
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
			//TODO: Refactor by making broadcast fn.
			for _, channel := range clients {
				go func(messageChannel chan<- []byte) {
					messageChannel <- []byte("TCP heartbeat from " + localAddress)
				}(channel)
			}
		}
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
