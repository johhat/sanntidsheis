package udp

import (
	"log"
	"net"
	"time"
)

const (
	broadcastAddress  = "255.255.255.255:10001"
	heartBeatInterval = 10
)

func recieve(recieveChan chan<- string, broadcastListener *net.UDPConn) {
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

		recieveChan <- address.IP.String()
	}
}

func broadcast(broadcastChan <-chan string, localListener *net.UDPConn) {

	addr, _ := net.ResolveUDPAddr("udp", broadcastAddress)

	for msg := range broadcastChan {
		_, err := localListener.WriteToUDP([]byte(msg), addr)

		if err != nil {
			log.Println(err)
		}
	}
}

func Init(udpHeartbeat chan string, localIp string) {

	addr, _ := net.ResolveUDPAddr("udp", ":10002")

	localListener, err := net.ListenUDP("udp", addr)

	if err != nil {
		log.Fatal(err)
	}

	addr, _ = net.ResolveUDPAddr("udp", broadcastAddress)

	broadcastListener, err := net.ListenUDP("udp", addr)

	if err != nil {
		log.Fatal(err)
	}

	broadcastChan := make(chan string)
	go broadcast(broadcastChan, localListener)

	recieveChan := make(chan string)
	go recieve(recieveChan, broadcastListener)

	log.Println("UDP initialized")

	for {
		select {
		case <-time.Tick(heartBeatInterval * time.Second):
			broadcastChan <- "Heartbeat"
		case msg := <-recieveChan:
			if msg != localIp {
				udpHeartbeat <- msg
			}
		}
	}
}
