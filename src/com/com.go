package com

import (
	"../networking"
	"../simdriver"
	"../manager"
	"encoding/json"
	"log"
)

type OrderAssignment struct {
	Button   simdriver.ClickEvent
	Assignee networking.IP
}

type Event struct{
	Type 	EventType
	Button  simdriver.ClickEvent
	NewState manager.State
}

type EventType int
const(
	NewExternalOrder EventType = iota
	NewInternalOrder
	SelfAssignedOrder
	PassingFloor
	DoorOpenedByInternalOrder
	StoppingToFinishOrder
	LeavingFloor
	SendState
)

func DecodeClientPacket(b []byte) (OrderAssignment, Event, error) {
	var oa OrderAssignment
	var ev Event
	var objmap map[string]*json.RawMessage

	err := json.Unmarshal(b, &objmap)
	if val, ok := objmap["OrderAssignment"]; ok {
    	
	} else if val, ok := objmap["Event"]; ok {

	}

	err := json.Unmarshal(b, &result)
	return result, err
}

func EncodeOrderAssignment(oa OrderAssignment) []byte {
	result, err := json.Marshal(oa)
	if err != nil {
		log.Fatal(err)
	}
	return result
}

func EncodeEvent(ev Event) []byte {
	result, err := json.Marshal(ev)
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
