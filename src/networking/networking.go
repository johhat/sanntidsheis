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

	//Mutable states
	status := true
	connectionStatuses := make(map[string]connectionStatus)
	heartbeats := make(map[string]int)
	clients := make(map[string]tcp.ClientInterface)

	//UDP
	var stopUdpHeartbeats chan<- bool
	udpBroadcastMsg, udpRecvMsg := udp.Init(localIp)
	stopUdpHeartbeats = udpSendHeartbeats(udpBroadcastMsg)

	//TCP
	tcpClient := make(chan tcp.ClientInterface)
	clientDisconnected := make(chan string)
	setTCPStatus := make(chan bool)
	tcpDial := make(chan tcp.DialRequest)
	tcpDialFail := make(chan string)

	go tcp.Init(tcpClient, tcpDial, setTCPStatus, localIp)

	for {
		select {
		case msg := <-sendMsgChan:
			handleTcpSendMsg(msg, clients)
		case rawMsg := <-udpRecvMsg:
			if status {
				handleUDPRecvMsg(rawMsg, connectionStatuses, heartbeats, tcpDial, tcpDialFail)
			}
		case ip := <-tcpDialFail:
			delete(connectionStatuses, ip)
			delete(heartbeats, ip)
		case newClient := <-tcpClient:
			status, ok := connectionStatuses[newClient.Ip]

			if ok && status == connected {
				//TODO: Tenk gjennom om dette er et scenario som kan skje - for i så fall er det ugreit
				log.Println("Network module add new client: Client allready registered as connected")
				continue
			}

			connectionStatuses[newClient.Ip] = connected
			clients[newClient.Ip] = newClient

			go handleTCPClient(newClient, recvMsgChan, connectedChan, clientDisconnected)

		case disconnectedClient := <-clientDisconnected:
			disconnectedChan <- disconnectedClient
			delete(clients, disconnectedClient)
			delete(connectionStatuses, disconnectedClient)
			delete(heartbeats, disconnectedClient)
		case newStatus := <-setStatus:
			if newStatus == status {
				log.Println("Tried to set network module to its current status", status)
				continue
			}

			status = newStatus

			setTCPStatus <- status

			if status {
				log.Println("Setting network module to active")
				stopUdpHeartbeats = udpSendHeartbeats(udpBroadcastMsg)
			} else {
				log.Println("Setting network module to inactive")
				close(stopUdpHeartbeats)
			}
		}
	}
}

func handleTCPClient(client tcp.ClientInterface,
	recvMsg chan<- com.Message,
	connectedChan,
	clientDisconnected chan<- string) {

	connectedChan <- client.Ip

	var recvFromClient <-chan tcp.RawMessage

	recvFromClient = client.RecvMsg

	for {
		select {
		case rawMsg, isOpen := <-recvFromClient:

			if !isOpen {
				recvFromClient = nil
				continue
			}

			decodedMsg, err := com.DecodeWrappedMessage(rawMsg.Data, rawMsg.Ip)

			if err != nil {
				log.Println("Error when decoding TCP msg:", err, string(rawMsg.Data), "Sender:", rawMsg.Ip)
				continue
			}

			recvMsg <- decodedMsg

		case <-client.IsDisconnected:
			for rawMsg := range client.RecvMsg {
				decodedMsg, err := com.DecodeWrappedMessage(rawMsg.Data, rawMsg.Ip)

				if err != nil {
					log.Println("Error when decoding TCP msg:", err, "Data:", string(rawMsg.Data), "Sender:", rawMsg.Ip)
					continue
				}

				recvMsg <- decodedMsg
			}

			clientDisconnected <- client.Ip
			log.Println("End of handleTCPCLient for ip", client.Ip)
			return
		}
	}
}

func handleTcpSendMsg(msg com.Message, clients map[string]tcp.ClientInterface) {

	switch msg := msg.(type) {
	case com.DirectedMessage:

		ip := msg.GetRecieverIp()

		client, ok := clients[ip]

		if !ok {
			log.Println("Error in handleTcpSendMsg. No client with ip:", ip)
			return
		}

		data, err := com.WrapMessage(msg).Encode()

		if err != nil {
			log.Println("Error when encoding msg in handleTcpSendMsg. Ignoring msg. Err:", err, "Msg:", msg)
			return
		}

		go func() {
			select {
			case <-client.IsDisconnected:
				log.Println("Failed to send msg. Client is disconnected. Msg:", string(data))
			case client.SendMsg <- data:
			}
		}()

	case com.Message:

		data, err := com.WrapMessage(msg).Encode()

		if err != nil {
			log.Println("Error when encoding msg in handleTcpSendMsg. Ignoring msg. Err:", err, "Msg:", msg)
			return
		}

		for _, client := range clients {
			go func(client tcp.ClientInterface) {
				select {
				case <-client.IsDisconnected:
					log.Println("Failed to broadcast msg to client. Client is disconnected. Msg:", string(data))
				case client.SendMsg <- data:
				}
			}(client)
		}

	default:
		log.Println("Error in handleTcpSendMsg: Message does not satisfy any relevant message interface")
	}
}

func handleUDPRecvMsg(rawMsg udp.RawMessage,
	connectionStatuses map[string]connectionStatus,
	heartbeats map[string]int,
	tcpDial chan<- tcp.DialRequest,
	tcpDialFail chan<- string) {

	m, err := com.DecodeWrappedMessage(rawMsg.Data, rawMsg.Ip)

	if err != nil {
		log.Println("Error when decoding udp msg:", err, string(rawMsg.Data))
		return
	}

	switch m := m.(type) {
	case com.Heartbeat:

		if m.Code != com.HeartbeatCode {
			log.Printf("Recieved heartbeat with invalid code. Valid code is %s while the heartbeat had code %s. Will not connect to client %s", com.HeartbeatCode, m.Code, rawMsg.Ip)
			return
		}

		if shouldDial(connectionStatuses, rawMsg.Ip, localIp) {
			connectionStatuses[rawMsg.Ip] = connecting
			go func() {
				req := tcp.DialRequest{
					Ip:          rawMsg.Ip,
					DialSuccess: make(chan bool),
				}
				tcpDial <- req

				result := <-req.DialSuccess

				if !result {
					tcpDialFail <- rawMsg.Ip
				}
			}()
		}

		registerHeartbeat(heartbeats, m.HeartbeatNum, rawMsg.Ip, "UDP")

	default:
		log.Println("Recieved and decoded non-heartbeat UDP message. Ignoring message.")
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
