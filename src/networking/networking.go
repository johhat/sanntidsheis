package networking

import (
	"../com"
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

	udpHeartbeatInterval = 1 * time.Second
	tcpHeartbeatInterval = 5 * time.Second
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
	tcpSendMsg := make(chan tcp.RawMessage)
	tcpBroadcastMsg := make(chan []byte)
	tcpRecvMsg := make(chan tcp.RawMessage)
	tcpConnected := make(chan string)
	tcpConnectionFailure := make(chan string)
	tcpDial := make(chan string)

	//TODO: Bryt tilkobling, Avbryt lytting og ikke gjør dial dersom frakoblet
	go tcp.Init(tcpSendMsg, tcpBroadcastMsg, tcpRecvMsg, tcpConnected, tcpConnectionFailure, tcpDial, localIp)
	go tcpSendHeartbeats(tcpBroadcastMsg, stopTcpHeartbeats) //TODO: Stopp utsending dersom frakoblet
	go handleTcpMsgRecv(tcpRecvMsg, recvMsgChan)

	for {
		select {
		case msg := <-sendMsgChan:
			handleTcpSendMsg(msg, clients, tcpSendMsg, tcpBroadcastMsg)
		case rawMsg := <-udpRecvMsg:
			//TODO: Ikke ring dersom frakoblet
			handleUdpMsgRecv(rawMsg, clients, tcpDial, localIp)
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
				//stopUdpHeartbeats <- true
				//stopTcpHeartbeats <- true
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
				// React on incoming TCP-calls
				// React on incoming HBs
				// Dial
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
		tcpSendMsg <- tcp.RawMessage{Data: w.Encode(), Ip: ip}

	case com.Message:
		w := com.WrapMessage(msg)
		tcpBroadcastMsg <- w.Encode()
	default:
		log.Println("Error in handleTcpSendMsg: Message does not satisfy any relevant message interface")
	}
}

func tcpSendHeartbeats(tcpBroadcastMsg chan<- []byte, quit <-chan bool) {

	tcpHeartbeatnum := 0
	tcpHeartbeatTick := time.Tick(tcpHeartbeatInterval)

	for {
		select {
		case <-tcpHeartbeatTick:
			m := com.CreateHeartbeat(tcpHeartbeatnum)
			w := com.WrapMessage(m)
			tcpBroadcastMsg <- w.Encode()
			tcpHeartbeatnum++
		case <-quit:
			return
		}
	}
}

func udpSendHeartbeats(udpBroadcastMsg chan<- []byte, quit <-chan bool) {

	udpHeartbeatNum := 0
	udpHeatbeatTick := time.Tick(udpHeartbeatInterval)

	for {
		select {
		case <-udpHeatbeatTick:
			m := com.CreateHeartbeat(udpHeartbeatNum)
			w := com.WrapMessage(m)
			udpBroadcastMsg <- w.Encode()
			udpHeartbeatNum++
		case <-quit:
			return
		}
	}
}

func handleTcpMsgRecv(tcpRecvMsg chan tcp.RawMessage, recvMsgChan chan<- com.Message) {

	heartbeats := make(map[string]int)

	for rawMsg := range tcpRecvMsg {
		m, err := com.DecodeWrappedMessage(rawMsg.Data, rawMsg.Ip)
		if err == nil {
			switch m.(type) {
			case com.Heartbeat:
				registerHeartbeat(heartbeats, m.(com.Heartbeat).HeartbeatNum, rawMsg.Ip, "TCP")
			default:
				recvMsgChan <- m
			}
		} else {
			log.Println("Error when decoding msg:", err, string(rawMsg.Data))
		}
	}
}

func registerHeartbeat(heartbeats map[string]int, heartbeatNum int, sender string, connectionType string) {

	prev, ok := heartbeats[sender]

	if !ok {
		heartbeats[sender] = heartbeatNum
		return
	} else {
		heartbeats[sender] = heartbeatNum
	}

	switch {
	case prev > heartbeatNum:
		log.Printf("Delayed %s heartbeat from %s. Previous HB: %v Current HB: %v \n", connectionType, sender, prev, heartbeatNum)
	case prev == heartbeatNum:
		log.Printf("Duplicate %s heartbeat from %s. Previous HB: %v Current HB: %v \n", connectionType, sender, prev, heartbeatNum)
	case prev+1 != heartbeatNum:
		log.Printf("Missing %s heartbeat(s) from %s. Previous HB: %v Current HB: %v \n", connectionType, sender, prev, heartbeatNum)
	}
}

func getUdpMsgRecvHandler() func(rawMsg udp.RawMessage, clients map[string]connectionStatus, tcpDial chan<- string, localIp string) {

	heartbeats := make(map[string]int) //Wrapped in closure in place of static variable

	return func(rawMsg udp.RawMessage, clients map[string]connectionStatus, tcpDial chan<- string, localIp string) {
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

func HasHighestIp(remoteIp string, localIp string) (bool, error) {

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

func GetLocalIp() string {
	return localIp
}
