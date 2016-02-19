package networking

import (
	"errors"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"tcp"
)

type connectionStatus int

const (
	connected connectionStatus = iota
	connecting
	disconnected
)

func NetworkLoop() {

	localIp, err := getLocalIp()

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Init network loop")

	clients := make(map[string]connectionStatus)

	udpHeartbeat := make(chan string)
	tcpMsg := make(chan string) //TODO: Bør inneholde ip og melding
	tcpConnected := make(chan string)
	tcpConnectionFailure := make(chan string)
	tcpDial := make(chan string)

	for {
		select {
		case remoteIp <- udpHeartbeat:
			if shouldDial(clients, remoteIp, localIp) {
				clients[remoteIp] = connecting
				tcpDial <- remoteIp
			}
		case remoteIp <- tcpConnected:
			clients[remoteIp] = connected
		case remoteIp <- tcpConnectionFailure:
			clients[remoteIp] = disconnected
		case msg <- tcpMsg:
			//Handle tcp msg here
			log.Println("TCP msg:", msg)
		}
	}
}

func shouldDial(clients map[string]connectionStatus, remoteIp string, localIp string) bool {
	status, ok = clients[remoteIp]

	if !ok {
		clients[remoteIp] = disconnected
	}

	if clients[remoteIp] == disconnected && hasHighestIp(remoteIp, localIp) {
		return true
	}

	return false
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
				return ipnet.IP.String()
			}
		}
	}

	return "", errors.New("Could not get local ip")
}

func hasHighestIp(remoteIp string, localIp string) bool {

	remoteIpInt, err1 := ipToInt(remoteIp)

	if err1 != nil {
		return 0, err1
	}

	localIpInt, err2 := ipToInt(localIp)

	if err2 != nil {
		return 0, err2
	}

	return remoteIpInt > localIpInt
}

func ipToInt(ip string) (int, errors) {
	ipParts := strings.SplitAfter(ip, ".")

	if len(ipParts) != 4 {
		return 0, errors.New("Malformed ip error")
	}

	return strconv.Atoi(ipParts[4])
}
