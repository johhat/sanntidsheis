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
	tcpListenPort  = ":6000"
	dialTriesLimit = 5

	readTimeout                   = 1 * time.Second
	heartbeatRate                 = readTimeout / 4
	heartbeatMessage              = "TCP-HEARTBEAT"
	heartbeatMessageRecvFormat    = heartbeatMessage + "\n"
	lenHeartbeatMessageRecvFormat = len(heartbeatMessageRecvFormat)
)

type ClientInterface struct {
	Ip             string
	SendMsg        chan []byte
	RecvMsg        chan RawMessage
	IsDisconnected chan bool
}

type DialRequest struct {
	Ip          string
	DialSuccess chan bool
}

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
	sendMsg      chan []byte
	recvMsg      chan RawMessage
	pipeIsClosed chan bool
	doDisconnect chan bool
}

func (c *client) RecieveFrom() <-chan bool {

	signalReturn := make(chan bool)

	go func() {
		defer func() {
			select {
			case signalReturn <- true:
			case <-c.chs.pipeIsClosed:
			}
			close(signalReturn)
			close(c.chs.recvMsg)
		}()

		reader := bufio.NewReader(c.conn)

		for {
			c.conn.SetReadDeadline(time.Now().Add(readTimeout))

			bytes, err := reader.ReadBytes('\n')

			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					return
				}				
				log.Println("TCP recv error. Connection: ", c.ip, " error:", err)
				return
			}

			if len(bytes) == lenHeartbeatMessageRecvFormat && string(bytes) == heartbeatMessageRecvFormat {
				continue
			}

			select {
			case c.chs.recvMsg <- RawMessage{Data: bytes, Ip: c.ip}:
			case <-c.chs.pipeIsClosed:
				return
			}
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
			case <-c.chs.pipeIsClosed:
			}
			close(signalReturn)
		}()

		var b bytes.Buffer

		for {
			select {
			case msg:=<- c.chs.sendMsg:

				b.Write(msg)
				b.WriteRune('\n')

				_, err := c.conn.Write(b.Bytes())

				b.Reset()

				if err != nil {
					if strings.Contains(err.Error(), "use of closed network connection") {
						return
					}
					log.Println("TCP send error. Connection: ", c.ip, " error:", err)
					return
				}

			case <-c.chs.pipeIsClosed:
				return
			}
		}
	}()
	return signalReturn
}

func handleClients(
	tcpClient chan<- ClientInterface,
	addClient <-chan client,
	rmClient <-chan client,
	closeAllConnections <-chan bool,
	localIp string) {

	clients := make(map[net.Conn]clientChans)

	for {
		select {
		case client := <-addClient:
			clients[client.conn] = client.chs
			tcpClient <- ClientInterface{
				Ip:             client.ip,
				SendMsg:        client.chs.sendMsg,
				RecvMsg:        client.chs.recvMsg,
				IsDisconnected: client.chs.pipeIsClosed,
			}
		case client := <-rmClient:
			delete(clients, client.conn)
		case <-closeAllConnections:
			for _, clientChans := range clients {
				clientChans.doDisconnect <- true
			}
		}
	}
}

func handleConnection(connection net.Conn, addClient chan<- client, rmClient chan<- client) {

	client := client{
		ip:   getRemoteIp(connection),
		conn: connection,
		chs: clientChans{
			sendMsg:      make(chan []byte),
			recvMsg:      make(chan RawMessage),
			pipeIsClosed: make(chan bool),
			doDisconnect: make(chan bool),
		},
	}

	addClient <- client

	mergeChanDone := make(chan bool)
	
	defer func() {
		close(client.chs.pipeIsClosed)
		close(mergeChanDone)
		connection.Close()
		rmClient <- client
	}()

	signalConnError := mergeChans(
		mergeChanDone,
		client.RecieveFrom(),
		client.SendTo())

	heartbeatTick := time.Tick(heartbeatRate)

	for {
		select {
		case <-heartbeatTick:
			client.chs.sendMsg <- []byte(heartbeatMessage)
		case <-signalConnError:
			return
		case <-client.chs.doDisconnect:
			return
		}
	}
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

func listen(addClient chan<- client, rmClient chan<- client, stopListener <-chan bool) {

	listener, err := net.Listen("tcp", tcpListenPort)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening for TCP connections on %v", listener.Addr())

	go func() {
		for {
			connection, err := listener.Accept()

			if err != nil {

				if strings.Contains(err.Error(), "use of closed network connection") {
					return
				} else {
					log.Println("Error in TCP listener:", err)
					continue
				}
			}

			log.Printf("Handling incoming connection from %v", connection.RemoteAddr())
			go handleConnection(connection, addClient, rmClient)
		}
	}()

	<-stopListener
	listener.Close()
}

func dial(request DialRequest, addClient chan<- client, rmClient chan<- client) {

	connection, err := net.Dial("tcp", request.Ip+tcpListenPort)

	numTries := 0

	for {
		if err != nil {
			log.Printf("TCP dial to %s failed", request.Ip+tcpListenPort)
			time.Sleep(500 * time.Millisecond) 
			connection, err = net.Dial("tcp", request.Ip+tcpListenPort)

			if numTries < dialTriesLimit {
				numTries++
				continue
			} else {
				request.DialSuccess <- false
				return
			}
		}

		log.Println("Handling dialed connection to ", request.Ip)
		go handleConnection(connection, addClient, rmClient)
		request.DialSuccess <- true
		return
	}
}

func getRemoteIp(connection net.Conn) string {
	return strings.Split(connection.RemoteAddr().String(), ":")[0]
}

func Init(tcpClient chan<- ClientInterface, tcpDial <-chan DialRequest,setStatus <-chan bool, localIp string) {

	status := true

	addClient := make(chan client)
	rmClient := make(chan client)

	stopListener := make(chan bool)
	closeAllConnections := make(chan bool)

	go handleClients(tcpClient, addClient, rmClient, closeAllConnections, localIp)
	go listen(addClient, rmClient, stopListener)

	for {
		select {
		case setTo := <-setStatus:

			if setTo == status {
				log.Println("TCP set to its current status", status)
				continue
			}

			status = setTo

			if setTo {
				go listen(addClient, rmClient, stopListener)
			} else {
				stopListener <- true
				closeAllConnections <- true
			}

		case request := <-tcpDial:
			if !status {
				log.Println("Abort dial as TCP module status is inactive")
				request.DialSuccess <- false
				continue
			}

			log.Println("Dialing ", request.Ip)
			go dial(request, addClient, rmClient)
		}
	}
}
