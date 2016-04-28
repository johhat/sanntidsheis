package main

import (
	"../com"
	"../networking"
	driver "../simdriver"
	"log"
	"time"
)

//Hovedpc på sanntid har ip 129.241.187.158

//For å teste på data til høyre på sanntidssalen
//I en terminal:  ssh student@129.241.187.159
//I en annen terminal, for å kopiere over repo: scp -r Documents/sanntidsheis/ student@129.241.187.159:Documents/

//For å teste på data bak til høyre på sanntidssalen
//I en terminal:  ssh student@129.241.187.161
//I en annen terminal, for å kopiere over repo: scp -r Documents/sanntidsheis/ student@129.241.187.161:Documents/

//Passordet i begge tilfeller: passord: Sanntid15

//Closed ch returnerer umiddelbart ved recv. Panic ved close.
//Nil ch blokkerer uendelig, både send og recieve
//For range stopper når en kanal er stengt
//Om pipelines: http://blog.golang.org/pipelines

func main() {

	log.Println("--Start of network main file--")

	sendMsgChan, recvMsgChan := make(chan com.Message), make(chan com.Message)
	connected, disconnected := make(chan string), make(chan string)
	disconnectFromNetwork, reconnectToNetwork := make(chan bool), make(chan bool)

	go networking.NetworkLoop(
		sendMsgChan,
		recvMsgChan,
		connected,
		disconnected,
		disconnectFromNetwork,
		reconnectToNetwork)

	go func() {
		for {
			<-time.Tick(1 * time.Second)
			//This is a broadcast msg
			sendMsgChan <- com.OrderAssignmentMsg{
				Button:   driver.ClickEvent{},
				Assignee: "192.168.0.0",
				Sender:   "192.168.0.100,",
			}
		}
	}()

	for {
		select {
		case msg := <-recvMsgChan:
			log.Println("Network main recieved message:", msg)
		case ip := <-connected:
			log.Println("IP connected", ip)
		case ip := <-disconnected:
			log.Println("IP disconnected", ip)
		}
	}
}
