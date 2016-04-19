package com

import (
	"../networking"
	"../simdriver"
	"encoding/json"
	"log"
)


type OrderAssignment struct {
	Button   simdriver.ClickEvent
	Assignee networking.IP
}

type State struct {
	LastPassedFloor int
	Direction simdriver.MotorDirection
	Orders
	SequenceNumber int
}

type Event struct{
	Type 	EventType
	Button  simdriver.ClickEvent
	NewState State
	
}

type EventType int
const(
	NewExternalOrder EventType = iota
	NewInternalOrder
	SelfAssignedOrder
	PassingFloor
	//DoorOpenedByInternalOrder
	StoppingToFinishOrder
	LeavingFloor

)

// EVENTS
//Ny ekstern ordre fra nett: etasje, retning
//No ordre intern knapp: etasje
//Tildelt ordre til seg selv: etasje, retning

//Kom til ny etasje og stopper ikke
//Dør åpnet av intern ordre i nåværende etasje
//Stopper i etasje og fullfører ordre
//Beveger seg ut av etasje

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
