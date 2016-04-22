package main

import (
	"../networking"
	"../networking/messages"
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

func main() {

	log.Println("--Start of network main file--")

	sendMsgChan := make(chan messages.Message)
	recvMsgChan := make(chan messages.Message)
	connected := make(chan string)
	disconnected := make(chan string)

	//m := messages.MockMessage{}
	//m.Number = 0
	//m.Text = "Hello from mock message!"

	m := messages.MockDirectedMessage{}
	m.Number = 0
	m.Text = "Hello from mock directed message!"
	m.Reciever = "129.241.187.159"

	go func() {
		for {
			<-time.Tick(5 * time.Second)
			m.Number++
			sendMsgChan <- m
		}
	}()

	go networking.NetworkLoop(sendMsgChan, recvMsgChan, connected, disconnected)

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
