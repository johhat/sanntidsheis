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
			<-time.Tick(1 * time.Second)
			m.Number = m.Number + 1
			sendMsgChan <- m
		}

	}()

	networking.NetworkLoop(sendMsgChan, recvMsgChan)
}
