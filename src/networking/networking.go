package networking

import (
	"../com"
	"./tcp"
	"./udp"
	"log"
	"time"
)

type connectionStatus int

const (
	connected connectionStatus = iota
	connecting
	disconnected

	udpHeartbeatInterval = 1 * time.Second
	tcpHeartbeatInterval = 250 * time.Millisecond
)

var localIp string

func init() {

	var err error

	localIp, err = getLocalIp()

	if err != nil {
		log.Fatal(err)
	}
}

func NetworkLoop(sendMsgChan <-chan com.Message,
	recvMsgChan chan<- com.Message,
	connectedChan,
	disconnectedChan chan<- string,
	disconnectFromNetwork,
	reconnectToNetwork <-chan bool) {

	log.Println("---Init network loop---")
	log.Println("The ip of this computer is: ", localIp)

	//Change state here
	networkModuleIsActive := true
	clients := make(map[string]connectionStatus)

	//Make collection of abortchans - for range over them on disconnectFromNetwork-signal
	stopUdpHeartbeats, stopTcpHeartbeats := make(chan bool), make(chan bool)

	//UDP
	udpBroadcastMsg := make(chan []byte)
	udpRecvMsg := make(chan udp.RawMessage)
	handleUdpMsgRecv := getUdpMsgRecvHandler()
	go udp.Init(udpBroadcastMsg, udpRecvMsg, localIp)
	go udpSendHeartbeats(udpBroadcastMsg, stopUdpHeartbeats) //TODO: Stopp utsending dersom frakoblet

	//TCP
	setTcpOperationMode := make(chan tcp.TcpOperationMode)

	tcpSendMsg := make(chan tcp.RawMessage)
	tcpBroadcastMsg := make(chan []byte)
	tcpRecvMsg := make(chan tcp.RawMessage)
	tcpConnected := make(chan string)
	tcpConnectionFailure := make(chan string)
	tcpDial := make(chan string)

	//TODO: Bryt tilkobling, Avbryt lytting og ikke gjør dial dersom frakoblet
	go tcp.Init(tcpSendMsg, tcpBroadcastMsg, tcpRecvMsg, tcpConnected, tcpConnectionFailure, tcpDial, setTcpOperationMode, localIp)
	go tcpSendHeartbeats(tcpBroadcastMsg, stopTcpHeartbeats)
	go handleTcpMsgRecv(tcpRecvMsg, recvMsgChan)

	for {
		select {
		case msg := <-sendMsgChan:
			handleTcpSendMsg(msg, clients, tcpSendMsg, tcpBroadcastMsg)
		case rawMsg := <-udpRecvMsg:
			if networkModuleIsActive {
				handleUdpMsgRecv(rawMsg, clients, tcpDial, localIp)
			}
		case remoteIp := <-tcpConnected:
			clients[remoteIp] = connected
			connectedChan <- remoteIp
		case remoteIp := <-tcpConnectionFailure:
			clients[remoteIp] = disconnected
			disconnectedChan <- remoteIp
		case <-disconnectFromNetwork:
			if networkModuleIsActive != false {
				networkModuleIsActive = false

				log.Println("disconnectFromNetwork is noop")
				//TODO: Handle disconnect
				// Stop outgoing HBs
				stopUdpHeartbeats <- true
				stopTcpHeartbeats <- true
				setTcpOperationMode <- tcp.Idle

				// Disconnect from existing TCP-conns

				// Ignore incoming TCP-dials
				// Ignore incoming HBs
				// Do not dial

			}

		case <-reconnectToNetwork:
			if networkModuleIsActive != true {

				log.Println("reconnectToNetwork is noop")
				//TODO: Handle reconnect
				// Restart outgoing HBs
				go udpSendHeartbeats(udpBroadcastMsg, stopUdpHeartbeats)
				go tcpSendHeartbeats(tcpBroadcastMsg, stopTcpHeartbeats)
				// React on incoming TCP-calls
				// React on incoming HBs
				// Dial
				setTcpOperationMode <- tcp.Active
				networkModuleIsActive = true
			}
		}
	}
}

func handleTcpSendMsg(msg com.Message, clients map[string]connectionStatus, tcpSendMsg chan<- tcp.RawMessage, tcpBroadcastMsg chan<- []byte) {

	switch msg.(type) {
	case com.DirectedMessage:
		directedMsg := msg.(com.DirectedMessage)

		ip := directedMsg.GetRecieverIp()

		status, ok := clients[ip]

		if status != connected || !ok {
			log.Println("Error in handleTcpSendMsg. Not connected to ip:", ip)
			return
		}

		w := com.WrapMessage(directedMsg)

		data, err := w.Encode()

		if err != nil {
			log.Println("Error when encoding msg in handleTcpSendMsg. Ignoring msg. Err:", err, "Msg:", msg)
			return
		}

		tcpSendMsg <- tcp.RawMessage{Data: data, Ip: ip}
	case com.Message:
		w := com.WrapMessage(msg)

		data, err := w.Encode()

		if err != nil {
			log.Println("Error when encoding msg in handleTcpSendMsg. Ignoring msg. Err:", err, "Msg:", msg)
			return
		}

		tcpBroadcastMsg <- data
	default:
		log.Println("Error in handleTcpSendMsg: Message does not satisfy any relevant message interface")
	}
}

func handleTcpMsgRecv(tcpRecvMsg chan tcp.RawMessage, recvMsgChan chan<- com.Message) {

	heartbeats := make(map[string]int)

	for rawMsg := range tcpRecvMsg {
		m, _, err := com.DecodeWrappedMessage(rawMsg.Data, rawMsg.Ip)
		if err == nil {
			switch m.(type) {
			case com.Heartbeat:
				registerHeartbeat(heartbeats, m.(com.Heartbeat).HeartbeatNum, rawMsg.Ip, "TCP")
			default:
				recvMsgChan <- m
			}
		} else {
			log.Println("Error when decoding TCP msg:", err, string(rawMsg.Data))
		}
	}
}

func getUdpMsgRecvHandler() func(rawMsg udp.RawMessage, clients map[string]connectionStatus, tcpDial chan<- string, localIp string) {

	heartbeats := make(map[string]int) //Wrapped in closure in place of static variable

	return func(rawMsg udp.RawMessage, clients map[string]connectionStatus, tcpDial chan<- string, localIp string) {
		m, _, err := com.DecodeWrappedMessage(rawMsg.Data, rawMsg.Ip)

		if err != nil {
			log.Println("Error when decoding udp msg:", err, string(rawMsg.Data))
		} else {
			switch m.(type) {
			case com.Heartbeat:

				if m.(com.Heartbeat).Code != com.HeartbeatCode {
					log.Printf("Recieved heartbeat with invalid code. Valid code is %s while the heartbeat had code %s. Will not connect to client %s", com.HeartbeatCode, m.(com.Heartbeat).Code, rawMsg.Ip)
					return
				}

				if shouldDial(clients, rawMsg.Ip, localIp) {
					clients[rawMsg.Ip] = connecting
					tcpDial <- rawMsg.Ip
				}

				registerHeartbeat(heartbeats, m.(com.Heartbeat).HeartbeatNum, rawMsg.Ip, "UDP")

			default:
				log.Println("Recieved and decoded non-heartbeat UDP message. Ignoring message.")
			}
		}
	}
}

func shouldDial(clients map[string]connectionStatus, remoteIp string, localIp string) bool {

	status, ok := clients[remoteIp]

	if !ok {
		clients[remoteIp] = disconnected
		status = disconnected
	}

	if status == disconnected {
		isHighest, err := HasHighestIp(remoteIp, localIp)

		if err != nil {
			log.Println(err)
			return false
		}

		return isHighest
	}

	return false
}
