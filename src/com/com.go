package com

// In progress

import (
	"../networking"
	"../simdriver"
	"encoding/json"
	"log"
)

type Order struct {
	Button   simdriver.OrderButton
	Assignee networking.ID
}

type State struct {
	LastPassedFloor int
	Requests        []Order
}

func DecodeClientPacket(b []byte) (ClientData, error) {
	var result ClientData
	err := json.Unmarshal(b, &result)
	return result, err
}

func EncodeMasterData(m MasterData) []byte {
	result, err := json.Marshal(m)
	if err != nil {
		log.Fatal(err)
	}
	return result
}

type LiftEvents struct {
	FloorReached chan int
	StopButton   chan bool
	Obstruction  chan bool
}
