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
	tcpListenPort    = ":6000"
	readTimeout      = 1 * time.Second
	heartbeatRate    = readTimeout / 4
	heartbeatMessage = "TCP-HEARTBEAT" //TODO: Declare as byte array
)

type ClientInterface struct {
	Ip             string
	SendMsg        chan []byte
	RecvMsg        chan RawMessage
	IsDisconnected chan bool
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
			log.Println("End of RecieveFrom from", c.ip)
		}()

		reader := bufio.NewReader(c.conn)

		for {
			c.conn.SetReadDeadline(time.Now().Add(readTimeout))

			bytes, err := reader.ReadBytes('\n')

			if err != nil {
				log.Println("TCP recv error. Connection: ", c.ip, " error:", err)
				return
			}

			//TODO: Test om den tar med \n nå den leser
			if str := string(bytes); str == heartbeatMessage || str == heartbeatMessage+"\n" {
				continue
			}

			c.chs.recvMsg <- RawMessage{Data: bytes, Ip: c.ip}
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
			log.Println("End of SendTo from", c.ip)
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

//TODO: Rename, consider removing.
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

	//TODO: Være sikker på at man unngår minnelekkasje ved at chs.disconnect ikke stenges

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

	addClient <- client //This must happen before rmClient, i.e. no go'ing

	defer func() {
		close(client.chs.pipeIsClosed)
		connection.Close()
		log.Printf("Connection from %v closed.\n", connection.RemoteAddr())
		rmClient <- client
		close(client.chs.recvMsg) //When RecieveFrom returns, there are no senders left
		close(client.chs.sendMsg) //Force panic in any go-routines blocked on send to client
	}()

	signalConnError := mergeChans(
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
			//TODO: Test if this is ok. Consider who should close
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
				log.Println("Error in TCP listener:", err)
				return //TODO: Return only if closed else continue
			}

			log.Printf("Handling incoming connection from %v", connection.RemoteAddr())
			go handleConnection(connection, addClient, rmClient)
		}
	}()

	<-stopListener
	listener.Close()
}

func dial(remoteIp string,
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
			go handleConnection(connection, addClient, rmClient)
			break
		}
	}
}

func getRemoteIp(connection net.Conn) string {
	//TODO: Consider adding error checking here
	return strings.Split(connection.RemoteAddr().String(), ":")[0]
}

//TODO: Change naming of setOnOff
func Init(tcpClient chan<- ClientInterface,
	tcpDial <-chan string,
	setOnOff <-chan bool,
	localIp string) {

	status := true

	addClient := make(chan client)
	rmClient := make(chan client)

	stopListener := make(chan bool)
	closeAllConnections := make(chan bool)

	go handleClients(tcpClient, addClient, rmClient, closeAllConnections, localIp)
	go listen(addClient, rmClient, stopListener)

	for {
		select {
		case setTo := <-setOnOff:

			if setTo == status {
				log.Println("TCP set to its current status", status)
				continue
			}

			status = setTo

			if setTo {
				log.Println("Setting TCP module to active")
				go listen(addClient, rmClient, stopListener)
			} else {
				log.Println("Setting TCP module to inactive")
				stopListener <- true
				closeAllConnections <- true
			}

		case remoteIp := <-tcpDial:
			if !status {
				log.Println("Abort dial as TCP module status is inactive")
				continue
			}

			log.Println("Dialing ", remoteIp)
			go dial(remoteIp, addClient, rmClient)
		}
	}
}

//Incoming signal to stop TCP comm
//Do not accept incoming dials (close accepts from listener)
//Do not dial
//Close pending connections
