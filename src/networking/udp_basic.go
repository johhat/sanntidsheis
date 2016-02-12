package main

import (
	"log"
	"net"
	"os"
	"time"
)

const (
	BROADCAST_ADDRESS = "255.255.255.255:10001"
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

		recieveChan <- string(buffer) + " from " + address.String()
	}
}

func broadcast(broadcastChan <-chan string, localListener *net.UDPConn) {

	addr, _ := net.ResolveUDPAddr("udp", BROADCAST_ADDRESS)

	for msg := range broadcastChan {
		_, err := localListener.WriteToUDP([]byte(msg), addr)

		if err != nil {
			log.Println(err)
		}

		log.Println("Broadcasted:", msg)
	}
}

func main() {

	addr, _ := net.ResolveUDPAddr("udp", ":10002")

	localListener, err := net.ListenUDP("udp", addr)

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	addr, _ = net.ResolveUDPAddr("udp", BROADCAST_ADDRESS)

	broadcastListener, err := net.ListenUDP("udp", addr)

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	broadcastChan := make(chan string)
	go broadcast(broadcastChan, localListener)

	recieveChan := make(chan string)
	go recieve(recieveChan, broadcastListener)

	log.Println("UDP initialized")

	for {
		select {
		case <-time.Tick(10 * time.Second):
			broadcastChan <- "Heartbeat"
		case msg := <-recieveChan:
			log.Println("Recieved:", msg)
		}
	}
}
