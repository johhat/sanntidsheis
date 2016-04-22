package networking

import (
	"./messages"
	"./tcp"
	"./udp"
	"errors"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

type connectionStatus int

const (
	connected connectionStatus = iota
	connecting
	disconnected
)

const (
	udpHeartbeatInterval = 1 * time.Second
	tcpHeartbeatInterval = 5 * time.Second
)

func NetworkLoop(sendMsgChan <-chan messages.Message, recvMsgChan chan<- messages.Message) {

	localIp, err := getLocalIp()

	if err != nil {
		log.Fatal(err)
	}

	log.Println("---Init network loop---")
	log.Println("The ip of this computer is: ", localIp)

	clients := make(map[string]connectionStatus)

	udpBroadcastMsg := make(chan []byte)
	udpRecvMsg := make(chan udp.RawMessage)
	go udp.Init(udpBroadcastMsg, udpRecvMsg, localIp)
	go udpSendHeartbeats(udpBroadcastMsg)

	tcpSendMsg := make(chan tcp.RawMessage)
	tcpBroadcastMsg := make(chan []byte)
	tcpRecvMsg := make(chan tcp.RawMessage)
	tcpConnected := make(chan string)
	tcpConnectionFailure := make(chan string)
	tcpDial := make(chan string)
	go tcp.Init(tcpSendMsg, tcpBroadcastMsg, tcpRecvMsg, tcpConnected, tcpConnectionFailure, tcpDial, localIp)
	go tcpSendHeartbeats(tcpBroadcastMsg)
	go handleTcpMsgRecv(tcpRecvMsg, recvMsgChan)

	for {
		select {
		case msg := <-sendMsgChan:
			handleTcpSendMsg(msg, tcpSendMsg, tcpBroadcastMsg)
		case rawMsg := <-udpRecvMsg:
			handleUdpMsgRecv(rawMsg, clients, tcpDial, localIp)
		case remoteIp := <-tcpConnected:
			clients[remoteIp] = connected
		case remoteIp := <-tcpConnectionFailure:
			clients[remoteIp] = disconnected
		}
	}
}

func handleTcpSendMsg(msg messages.Message, tcpSendMsg chan<- tcp.RawMessage, tcpBroadcastMsg chan<- []byte) {

	switch msg.(type) {
	case messages.DirectedMessage:
		directedMsg := msg.(messages.DirectedMessage)
		w := messages.WrapMessage(directedMsg)
		tcpSendMsg <- tcp.RawMessage{Data: w.Encode(), Ip: directedMsg.GetRecieverIp()}
	case messages.Message:
		log.Println("This is a broadcast message", msg)
		w := messages.WrapMessage(msg)
		tcpBroadcastMsg <- w.Encode()
	default:
		log.Println("Error in handleTcpSendMsg: Message does not satisfy any relevant message interface")
	}
}

func tcpSendHeartbeats(tcpBroadcastMsg chan<- []byte) {

	tcpHeartbeatnum := 0
	tcpHeartbeatTick := time.Tick(tcpHeartbeatInterval)

	for {
		<-tcpHeartbeatTick
		m := messages.CreateHeartbeat(tcpHeartbeatnum)
		w := messages.WrapMessage(m)
		tcpBroadcastMsg <- w.Encode()
		tcpHeartbeatnum++
	}
}

func udpSendHeartbeats(udpBroadcastMsg chan<- []byte) {

	udpHeartbeatNum := 0
	udpHeatbeatTick := time.Tick(udpHeartbeatInterval)

	for {
		<-udpHeatbeatTick
		m := messages.CreateHeartbeat(udpHeartbeatNum)
		w := messages.WrapMessage(m)
		udpBroadcastMsg <- w.Encode()
		udpHeartbeatNum++
	}
}

func handleTcpMsgRecv(tcpRecvMsg chan tcp.RawMessage, recvMsgChan chan<- messages.Message) {

	//clientHeartbeatNum := make(map[string]int)

	for rawMsg := range tcpRecvMsg {
		m, err := messages.DecodeWrappedMessage(rawMsg.Data, rawMsg.Ip)
		if err == nil {
			switch m.(type) {
			case messages.Heartbeat:
				//TODO: Add check of HB-num. Detect missing.
			default:
				recvMsgChan <- m
			}
		} else {
			log.Println("Error when decoding msg:", err, string(rawMsg.Data))
		}
	}
}

func handleUdpMsgRecv(rawMsg udp.RawMessage, clients map[string]connectionStatus, tcpDial chan<- string, localIp string) {

	//TODO: Consider logging heartbeat-number
	//TODO: Check if heartbeat code is valid

	m, err := messages.DecodeWrappedMessage(rawMsg.Data, rawMsg.Ip)

	if err != nil {
		log.Println("Error when decoding udp msg:", err, string(rawMsg.Data))
	} else {
		switch m.(type) {
		case messages.Heartbeat:
			if shouldDial(clients, rawMsg.Ip, localIp) {
				clients[rawMsg.Ip] = connecting
				log.Println("TCP-connecting", clients)
				tcpDial <- rawMsg.Ip
			}
		default:
			log.Println("Recieved and decoded non-heartbeat UDP message. Ignoring message.")
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
		isHighest, err := hasHighestIp(remoteIp, localIp)

		if err != nil {
			log.Println(err)
			return false
		}

		return isHighest
	}

	return false
}

func hasHighestIp(remoteIp string, localIp string) (bool, error) {

	remoteIpInt, err1 := ipToInt(remoteIp)

	if err1 != nil {
		return false, err1
	}

	localIpInt, err2 := ipToInt(localIp)

	if err2 != nil {
		return false, err2
	}

	return remoteIpInt > localIpInt, nil
}

func ipToInt(ip string) (int, error) {
	ipParts := strings.SplitAfter(ip, ".")

	if len(ipParts) != 4 {
		//TODO: Return string with ip
		return 0, errors.New("Malformed ip error")
	}

	return strconv.Atoi(ipParts[3])
}

func getLocalIp() (string, error) {

	//TODO: Denne er copy paste fra SO. Bør kanskje kontrolleres.

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", errors.New("Could not get local ip")
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", errors.New("Could not get local ip")
}
