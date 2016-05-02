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

	udpHeartbeatInterval = 1 * time.Second
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
	setStatus <-chan bool) {

	log.Println("---Init network loop---")
	log.Println("The ip of this computer is: ", localIp)

	//Change state here
	status := true
	connectionStatuses := make(map[string]connectionStatus)
	tcpSendChans := make(map[string]chan<- []byte)

	//UDP
	stopUdpHeartbeats := make(chan bool)
	udpBroadcastMsg := make(chan []byte)
	udpRecvMsg := make(chan udp.RawMessage)
	handleUdpMsgRecv := getUdpMsgRecvHandler()
	go udp.Init(udpBroadcastMsg, udpRecvMsg, localIp)
	go udpSendHeartbeats(udpBroadcastMsg, stopUdpHeartbeats) //TODO: Stopp utsending dersom frakoblet

	//TCP
	tcpClient := make(chan tcp.ClientInterface)
	clientDisconnected := make(chan string)
	setTCPStatus := make(chan bool)
	tcpDial := make(chan string)

	//TODO: Bryt tilkobling, Avbryt lytting og ikke gjør dial dersom frakoblet
	go tcp.Init(tcpClient, tcpDial, setTCPStatus, localIp)

	for {
		select {
		case msg := <-sendMsgChan:
			handleTcpSendMsg(msg, tcpSendChans)
		case rawMsg := <-udpRecvMsg:
			if status {
				handleUdpMsgRecv(rawMsg, connectionStatuses, tcpDial, localIp)
			}
		case newClient := <-tcpClient:
			connectionStatuses[newClient.Ip] = connected
			tcpSendChans[newClient.Ip] = newClient.SendMsg
			go handleTCPClient(newClient, recvMsgChan, connectedChan, clientDisconnected)
		case disconnectedClient := <-clientDisconnected:
			disconnectedChan <- disconnectedClient
			delete(tcpSendChans, disconnectedClient)
			delete(connectionStatuses, disconnectedClient)
		case newStatus := <-setStatus:
			if newStatus == status {
				log.Println("Tried to set network module to its current status", status)
				continue
			}

			status = newStatus

			if status {
				log.Println("Setting network module to active")
				go udpSendHeartbeats(udpBroadcastMsg, stopUdpHeartbeats)
			} else {
				log.Println("Setting network module to inactive")
				stopUdpHeartbeats <- true
			}
		}
	}
}

func handleTCPClient(client tcp.ClientInterface, recvMsg chan<- com.Message, connectedChan, clientDisconnected chan<- string) {
	//Make shure manager knows it is connected
	//Start recieving messages
	//Add sendChan to some sort of fan-out function
	//If disconnected is signaled, flush recvChan, signal manager, stop incoming messages, delete from array of connected

	connectedChan <- client.Ip

	for {
		select {
		case rawMsg := <-client.RecvMsg:
			decodedMsg, err := com.DecodeWrappedMessage(rawMsg.Data, rawMsg.Ip)

			if err != nil {
				log.Println("Error when decoding TCP msg:", err, string(rawMsg.Data), "Sender:", rawMsg.Ip)
				continue
			}

			recvMsg <- decodedMsg

		case <-client.IsDisconnected:
			//Flush incoming messages - the channel is closed when handleconnection in TCP returns
			for rawMsg := range client.RecvMsg {
				//Sending them to a blocking channel in manager here ensures all messages are read before manager gets disconn signal
				decodedMsg, err := com.DecodeWrappedMessage(rawMsg.Data, rawMsg.Ip)

				if err != nil {
					log.Println("Error when decoding TCP msg:", err, string(rawMsg.Data), "Sender:", rawMsg.Ip)
					continue
				}

				recvMsg <- decodedMsg
			}
			//Signal disconnect to manager
			clientDisconnected <- client.Ip
			return
		}
	}
}

func handleTcpSendMsg(msg com.Message, sendChans map[string]chan<- []byte) {

	switch msg := msg.(type) {
	case com.DirectedMessage:

		ip := msg.GetRecieverIp()

		sendChan, ok := sendChans[ip]

		if !ok {
			log.Println("Error in handleTcpSendMsg. No sendChan to ip:", ip)
			return
		}

		data, err := com.WrapMessage(msg).Encode()

		if err != nil {
			log.Println("Error when encoding msg in handleTcpSendMsg. Ignoring msg. Err:", err, "Msg:", msg)
			return
		}

		sendChan <- data //TODO: Must have a disconnected path. I.e. pipe closed.

	case com.Message:

		data, err := com.WrapMessage(msg).Encode()

		if err != nil {
			log.Println("Error when encoding msg in handleTcpSendMsg. Ignoring msg. Err:", err, "Msg:", msg)
			return
		}

		//Broadcast
		for _, sendChan := range sendChans {
			sendChan <- data //TODO: Must have a disconnected path. I.e. pipe closed.
		}

	default:
		log.Println("Error in handleTcpSendMsg: Message does not satisfy any relevant message interface")
	}
}

func getUdpMsgRecvHandler() func(rawMsg udp.RawMessage, connectionStatuses map[string]connectionStatus, tcpDial chan<- string, localIp string) {

	heartbeats := make(map[string]int) //Wrapped in closure in place of static variable

	return func(rawMsg udp.RawMessage, connectionStatuses map[string]connectionStatus, tcpDial chan<- string, localIp string) {
		m, err := com.DecodeWrappedMessage(rawMsg.Data, rawMsg.Ip)

		if err != nil {
			log.Println("Error when decoding udp msg:", err, string(rawMsg.Data))
		} else {
			switch m.(type) {
			case com.Heartbeat:

				if m.(com.Heartbeat).Code != com.HeartbeatCode {
					log.Printf("Recieved heartbeat with invalid code. Valid code is %s while the heartbeat had code %s. Will not connect to client %s", com.HeartbeatCode, m.(com.Heartbeat).Code, rawMsg.Ip)
					return
				}

				if shouldDial(connectionStatuses, rawMsg.Ip, localIp) {
					connectionStatuses[rawMsg.Ip] = connecting
					tcpDial <- rawMsg.Ip
				}

				registerHeartbeat(heartbeats, m.(com.Heartbeat).HeartbeatNum, rawMsg.Ip, "UDP")

			default:
				log.Println("Recieved and decoded non-heartbeat UDP message. Ignoring message.")
			}
		}
	}
}

func shouldDial(connectionStatuses map[string]connectionStatus, remoteIp string, localIp string) bool {

	_, ok := connectionStatuses[remoteIp]

	if !ok {
		isHighest, err := HasHighestIp(remoteIp, localIp)

		if err != nil {
			log.Println(err)
			return false
		}

		return isHighest
	}

	return false
}
