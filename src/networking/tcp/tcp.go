package tcp

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

type RawMessage struct {
	Data []byte
	Ip   string
}

type client struct {
	ip   string
	conn net.Conn
	ch   chan []byte
}

func (c client) RecieveFrom(ch chan<- RawMessage, closeConnection chan<- bool) {

	reader := bufio.NewReader(c.conn)

	for {
		c.conn.SetReadDeadline(time.Now().Add(writeTimeout))

		bytes, err := reader.ReadBytes('\n')

		if err != nil {
			log.Println("TCP recv error. Connection: ", c.ip, " error:", err)
			closeConnection <- true
			return
		}
		ch <- RawMessage{Data: bytes, Ip: c.ip}
	}
}

func (c client) SendTo(ch <-chan []byte, closeConnection chan<- bool) {

	var b bytes.Buffer

	for msg := range ch {

		b.Write(msg)
		b.WriteRune('\n')

		_, err := c.conn.Write(b.Bytes())

		b.Reset()

		if err != nil {
			log.Println("TCP send error. Connection: ", c.ip, " error:", err)
			closeConnection <- true
			return
		}
	}
}

func handleMessages(sendMsg <-chan RawMessage, broadcastMsg <-chan []byte, addClient <-chan client, rmClient <-chan client, localIp string, tcpConnected chan string, tcpConnectionFailure chan string) {

	clients := make(map[net.Conn]chan<- []byte)

	for {
		select {
		case rawMsg := <-sendMsg:
			sendToIp(rawMsg.Ip, clients, rawMsg.Data)
		case msg := <-broadcastMsg:
			broadcast(clients, msg)
		case client := <-addClient:
			clients[client.conn] = client.ch
			tcpConnected <- client.ip
		case client := <-rmClient:
			delete(clients, client.conn)
			tcpConnectionFailure <- client.ip
		}
	}
}

func sendToIp(ip string, clients map[net.Conn]chan<- []byte, message []byte) {
	for connection, channel := range clients {
		if getRemoteIp(connection) == ip {
			channel <- message
			return
		}
	}
	log.Println("TCP send to ip failed. No existing connection to ip", ip)
}

func broadcast(clients map[net.Conn]chan<- []byte, message []byte) {
	for _, channel := range clients {
		go func(messageChannel chan<- []byte) {
			messageChannel <- message
		}(channel)
	}
}

func handleConnection(connection net.Conn, recvMsg chan<- RawMessage, addClient chan<- client, rmClient chan<- client) {

	defer connection.Close()

	client := client{
		ip:   getRemoteIp(connection),
		conn: connection,
		ch:   make(chan []byte),
	}

	addClient <- client

	defer func() {
		log.Printf("Connection from %v closed.\n", connection.RemoteAddr())
		rmClient <- client
	}()

	closeConnection := make(chan bool)

	go client.RecieveFrom(recvMsg, closeConnection)
	go client.SendTo(client.ch, closeConnection)

	<-closeConnection
}

func listen(recvMsg chan<- RawMessage, addClient chan<- client, rmClient chan<- client) {

	listener, err := net.Listen("tcp", tcpPort)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening for TCP connections on %v", listener.Addr())

	for {
		connection, err := listener.Accept()

		if err != nil {
			log.Println("Errpr in TCP listener:", err)
			continue
		}
		log.Printf("Handling incoming connection from %v", connection.RemoteAddr())
		go handleConnection(connection, recvMsg, addClient, rmClient)
	}
}

func dial(remoteIp string, recvMsg chan<- RawMessage, addClient chan<- client, rmClient chan<- client) {
	connection, err := net.Dial("tcp", remoteIp+tcpPort)

	for {
		if err != nil {
			log.Printf("TCP dial to %s failed", remoteIp+tcpPort)
			time.Sleep(500 * time.Millisecond) //TODO: Avslutte etter et visst antall forsøk? Må i så fall gi beskjed til modul om fail
			connection, err = net.Dial("tcp", remoteIp+tcpPort)
		} else {
			log.Println("Handling dialed connection to ", remoteIp)
			go handleConnection(connection, recvMsg, addClient, rmClient)
			break
		}
	}
}

func getRemoteIp(connection net.Conn) string {
	//TODO: Consider adding error checking here
	return strings.Split(connection.RemoteAddr().String(), ":")[0]
}

func Init(tcpSendMsg <-chan RawMessage, tcpBroadcastMsg <-chan []byte, tcpRecvMsg chan RawMessage, tcpConnected, tcpConnectionFailure, tcpDial chan string, localIp string) {

	addClient := make(chan client)
	rmClient := make(chan client)

	go handleMessages(tcpSendMsg, tcpBroadcastMsg, addClient, rmClient, localIp, tcpConnected, tcpConnectionFailure)
	go listen(tcpRecvMsg, addClient, rmClient)

	for {
		remoteIp := <-tcpDial
		log.Println("Dialing ", remoteIp)
		go dial(remoteIp, tcpRecvMsg, addClient, rmClient)
	}
}
