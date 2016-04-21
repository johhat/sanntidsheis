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
	tcpHeartbeatInterval = 5 * time.Second
)

func NetworkLoop(sendMsgChan chan messages.Message, recvMsgChan chan<- messages.Message) {

	localIp, err := getLocalIp()

	log.Println("---Init network loop---")
	log.Println("The ip of this computer is: ", localIp)

	if err != nil {
		log.Fatal(err)
	}

	clients := make(map[string]connectionStatus)

	udpHeartbeat := make(chan string)

	tcpSendMsg := make(chan tcp.RawMessage)
	tcpBroadcastMsg := make(chan []byte)
	tcpRecvMsg := make(chan tcp.RawMessage)
	tcpConnected := make(chan string)
	tcpConnectionFailure := make(chan string)
	tcpDial := make(chan string)

	go udp.Init(udpHeartbeat, localIp)
	go tcp.Init(tcpSendMsg, tcpBroadcastMsg, tcpRecvMsg, tcpConnected, tcpConnectionFailure, tcpDial, localIp)

	tcpHeartbeatTick := time.Tick(tcpHeartbeatInterval)

	for {
		select {
		case msg := <-sendMsgChan:
			w := messages.WrapMessage(msg)
			tcpBroadcastMsg <- w.Encode()
		case remoteIp := <-udpHeartbeat:
			if shouldDial(clients, remoteIp, localIp) {
				clients[remoteIp] = connecting
				tcpDial <- remoteIp
			}
		case remoteIp := <-tcpConnected:
			clients[remoteIp] = connected
		case remoteIp := <-tcpConnectionFailure:
			clients[remoteIp] = disconnected
		case rawMsg := <-tcpRecvMsg: //TODO: Legg inn håndtering av meldinger her
			log.Println("Incoming TCP msg: \n", string(rawMsg.Data))
			m, err := messages.DecodeWrappedMessage(rawMsg.Data)
			log.Println("Decoded msg:", m)
			if err != nil {
				log.Println(err)
			}
		case <-tcpHeartbeatTick:
			sendMsgChan <- messages.CreateHeartbeat()
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
