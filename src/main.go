package main

import (
	"./networking"
	"./networking/messages"
)

func main() {

	// Start driver
	// Start elevator
	// Start queue

	// Start networking
	// - Output
	sendMsgChan := make(chan messages.Message)
	recvMsgChan := make(chan messages.Message)
	connected := make(chan string)
	disconnected := make(chan string)

	go networking.NetworkLoop(sendMsgChan, recvMsgChan, connected, disconnected)

	// Start manager
	// - Input:

}
