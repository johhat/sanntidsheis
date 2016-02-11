package main

//Source: http://synflood.at/tmp/golang-slides/mrmcd2012.html
//Test fra shell: echo -n "Random string" | nc localhost 6000

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

type Client struct {
	id   string
	conn net.Conn
	ch   chan string
}

func (c Client) RecieveFrom(ch chan<- string) {
	bufc := bufio.NewReader(c.conn)
	for {
		line, err := bufc.ReadString('\n')
		if err != nil {
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

func handleMessages(msgchan <-chan string, addchan <-chan Client, rmchan <-chan Client) {
	clients := make(map[net.Conn]chan<- string)

	for {
		select {
		case msg := <-msgchan: //Broadcast on TCP
			for _, ch := range clients {
				go func(mch chan<- string) {
					mch <- "\033[1;33;30m" + msg + "\033[m\r\n"
				}(ch)
			}
		case client := <-addchan: //Add client to list
			clients[client.conn] = client.ch
		case client := <-rmchan: //Remove client from list
			log.Printf("Client disconnects: %s\n", client.id)
			delete(clients, client.conn)
		case <-time.Tick(10 * time.Second): //Send heartbeat on TCP
			for _, ch := range clients {
				go func(mch chan<- string) {
					mch <- "\033[1;33;30m" + "Heartbeat from server" + "\033[m\r\n"
				}(ch)
			}
		}
	}
}

func handleConnection(connection net.Conn, msgchan chan<- string, addchan chan<- Client, rmchan chan<- Client) {

	defer connection.Close()

	client := Client{
		id:   connection.RemoteAddr().String(),
		conn: connection,
		ch:   make(chan string),
	}

	addchan <- client

	defer func() {
		msgchan <- fmt.Sprintf("Client %s disconnected.\n", client.id)
		log.Printf("Connection from %v closed.\n", connection.RemoteAddr())
		rmchan <- client
	}()

	io.WriteString(connection, fmt.Sprintf("Welcome, %s!\n\n", client.id))
	msgchan <- fmt.Sprintf("New user %s has joined the chat room.\n", client.id)

	// I/O
	go client.RecieveFrom(msgchan)
	client.SendTo(client.ch) //This blocks
}

func main() {
	listener, err := net.Listen("tcp", ":6000")

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	msgchan := make(chan string)
	addchan := make(chan Client)
	rmchan := make(chan Client)

	go handleMessages(msgchan, addchan, rmchan)

	log.Printf("Listening for TCP connections on %v", listener.Addr())

	for {
		connection, err := listener.Accept()

		if err != nil {
			log.Println(err)
			continue
		}

		go handleConnection(connection, msgchan, addchan, rmchan)
	}
}
