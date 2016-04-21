package udp

import (
	"log"
	"net"
)

const (
	broadcastAddress = "255.255.255.255:10001"
	listenPort       = ":10002"
)

type RawMessage struct {
	Data []byte
	Ip   string
}

func recieve(recieveChan chan<- RawMessage, broadcastListener *net.UDPConn) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Error in UDP recieve: %s \n Closing connection.", r)
			broadcastListener.Close()
		}
	}()

	for {
		buffer := make([]byte, 1024)
		n, address, err := broadcastListener.ReadFromUDP(buffer)

		if err != nil || n < 0 {
			log.Printf("Error in UDP recieve\n")
			panic(err)
		}

		recieveChan <- RawMessage{Data: buffer, Ip: address.IP.String()}
	}
}

func broadcast(broadcastChan <-chan []byte, localListener *net.UDPConn) {

	addr, _ := net.ResolveUDPAddr("udp", broadcastAddress)

	for msg := range broadcastChan {
		_, err := localListener.WriteToUDP(msg, addr)

		if err != nil {
			log.Println(err)
		}
	}
}

func Init(udpBroadcastMsg <-chan []byte, udpRecvMsg chan<- RawMessage, localIp string) {

	addr, _ := net.ResolveUDPAddr("udp", listenPort)

	localListener, err := net.ListenUDP("udp", addr)

	if err != nil {
		log.Fatal(err)
	}

	addr, _ = net.ResolveUDPAddr("udp", broadcastAddress)

	broadcastListener, err := net.ListenUDP("udp", addr)

	if err != nil {
		log.Fatal(err)
	}

	broadcastChan := make(chan []byte)
	go broadcast(broadcastChan, localListener)

	recieveChan := make(chan RawMessage)
	go recieve(recieveChan, broadcastListener)

	log.Println("UDP initialized")

	for {
		select {
		case msg := <-udpBroadcastMsg:
			broadcastChan <- msg
		case rawMsg := <-recieveChan:
			if rawMsg.Ip != localIp {
				udpRecvMsg <- rawMsg
			}
		}
	}
}
