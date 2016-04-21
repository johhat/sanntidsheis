package main

import (
	"../networking"
	"../networking/messages"
	"log"
	"time"
)

//For å teste på data til høyre på sanntidssalen
//I en terminal:  ssh student@129.241.187.159, passord: Sanntid15
//I en annen terminal, for å kopiere over repo:

func main() {

	log.Println("--Start of network main file--")

	sendMsgChan := make(chan messages.Message)
	recvMsgChan := make(chan messages.Message)

	m := messages.MockMessage{}

	m.Number = 0
	m.Text = "Hello from mock message!"

	go func() {
		for {
			<-time.Tick(5 * time.Second)
			m.Number++
			sendMsgChan <- m
		}

	}()

	go networking.NetworkLoop(sendMsgChan, recvMsgChan)

	for {
		msg := <-recvMsgChan
		log.Println("Network main recieved message:", msg)
	}
}
