package tcp

import (
	"bufio"
	"bytes"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	tcpListenPort = ":6000"
	readTimeout   = 10 * time.Second
)

type TcpOperationMode int

const (
	Active TcpOperationMode = iota
	Idle
)

type RawMessage struct {
	Data []byte
	Ip   string
}

type client struct {
	ip   string
	conn net.Conn
	chs  clientChans
}

type clientChans struct {
	sendMsg    chan []byte
	pipeClosed chan bool
	disconnect chan bool
}

func (c client) RecieveFrom(ch chan<- RawMessage) <-chan bool {

	signalReturn := make(chan bool)

	go func() {
		defer func() {
			select {
			case signalReturn <- true:
			case <-c.chs.pipeClosed:
			}
			close(signalReturn)
		}()

		reader := bufio.NewReader(c.conn)

		for {
			c.conn.SetReadDeadline(time.Now().Add(readTimeout))

			bytes, err := reader.ReadBytes('\n')

			if err != nil {
				log.Println("TCP recv error. Connection: ", c.ip, " error:", err)
				return
			}
			ch <- RawMessage{Data: bytes, Ip: c.ip}
		}
	}()
	return signalReturn
}

func (c *client) SendTo() <-chan bool {

	signalReturn := make(chan bool)

	go func() {
		defer func() {
			select {
			case signalReturn <- true:
			case <-c.chs.pipeClosed:
			}
			close(signalReturn)
		}()

		var b bytes.Buffer

		for msg := range c.chs.sendMsg {

			b.Write(msg)
			b.WriteRune('\n')

			_, err := c.conn.Write(b.Bytes())

			b.Reset()

			if err != nil {
				log.Println("TCP send error. Connection: ", c.ip, " error:", err)
				return
			}
		}
	}()
	return signalReturn
}

func handleMessages(sendMsg <-chan RawMessage,
	broadcastMsg <-chan []byte,
	addClient <-chan client,
	rmClient <-chan client,
	localIp string,
	tcpConnected chan<- string,
	tcpConnectionFailure chan<- string) {

	clients := make(map[net.Conn]clientChans)

	for {
		select {
		case rawMsg := <-sendMsg:
			go sendToIp(rawMsg.Ip, clients, rawMsg.Data)
		case msg := <-broadcastMsg:
			go broadcast(clients, msg)
		case client := <-addClient:
			clients[client.conn] = client.chs
			tcpConnected <- client.ip
		case client := <-rmClient:
			delete(clients, client.conn)
			tcpConnectionFailure <- client.ip
		}
	}
}

func sendToIp(ip string, clients map[net.Conn]clientChans, message []byte) {
	for connection, channels := range clients {
		if getRemoteIp(connection) == ip {
			select {
			case channels.sendMsg <- message:
			case <-channels.pipeClosed:
				log.Println("SendToIp send failed - pipe closed. Ip:", ip)
			}
			return
		}
	}
	log.Println("TCP send to ip failed. No existing connection to ip", ip)
}

func broadcast(clients map[net.Conn]clientChans, message []byte) {
	for _, chs := range clients {
		go func(chs clientChans) {
			select {
			case chs.sendMsg <- message:
			case <-chs.pipeClosed:
				log.Println("Broadcast to one recvr failed - pipe closed.")
			}
		}(chs)
	}
}

func handleConnection(connection net.Conn, recvMsg chan<- RawMessage, addClient chan<- client, rmClient chan<- client) {

	client := client{
		ip:   getRemoteIp(connection),
		conn: connection,
		chs: clientChans{
			sendMsg:    make(chan []byte),
			pipeClosed: make(chan bool),
			disconnect: make(chan bool),
		},
	}

	addClient <- client

	defer func() {
		close(client.chs.pipeClosed)
		connection.Close()
		log.Printf("Connection from %v closed.\n", connection.RemoteAddr())
		rmClient <- client
		close(client.chs.sendMsg) //Force panic in any go-routines blocked on send to client
	}()

	signalCloseConn := mergeChans(
		client.chs.pipeClosed,
		client.RecieveFrom(recvMsg),
		client.SendTo(),
		client.chs.disconnect)

	<-signalCloseConn
}

func mergeChans(done <-chan bool, channels ...<-chan bool) <-chan bool {
	var wg sync.WaitGroup
	merged := make(chan bool)

	output := func(ch <-chan bool) {
		defer wg.Done()
		for elem := range ch {
			select {
			case merged <- elem:
			case <-done:
				return
			}
		}
	}

	wg.Add(len(channels))
	for _, ch := range channels {
		go output(ch)
	}

	go func() {
		wg.Wait()
		close(merged)
	}()

	return merged
}

func listen(recvMsg chan<- RawMessage,
	addClient chan<- client,
	rmClient chan<- client) {

	listener, err := net.Listen("tcp", tcpListenPort)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening for TCP connections on %v", listener.Addr())

	for {
		connection, err := listener.Accept()

		if err != nil {
			log.Println("Error in TCP listener:", err)
			continue
		}
		log.Printf("Handling incoming connection from %v", connection.RemoteAddr())
		go handleConnection(connection, recvMsg, addClient, rmClient)
	}
}

func dial(remoteIp string,
	recvMsg chan<- RawMessage,
	addClient chan<- client,
	rmClient chan<- client) {
	connection, err := net.Dial("tcp", remoteIp+tcpListenPort)

	for {
		if err != nil {
			log.Printf("TCP dial to %s failed", remoteIp+tcpListenPort)
			time.Sleep(500 * time.Millisecond) //TODO: Avslutte etter et visst antall forsøk? Må i så fall gi beskjed til modul om fail
			connection, err = net.Dial("tcp", remoteIp+tcpListenPort)
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

func Init(tcpSendMsg <-chan RawMessage,
	tcpBroadcastMsg <-chan []byte,
	tcpRecvMsg chan<- RawMessage,
	tcpConnected,
	tcpConnectionFailure chan<- string,
	tcpDial <-chan string,
	setOperationMode <-chan TcpOperationMode,
	localIp string) {

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

//Incoming signal to stop TCP comm
//Do not accept incoming dials (close accepts from listener)
//Do not dial
//Close pending connections
