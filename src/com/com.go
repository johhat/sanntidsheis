package com

import (
	"../manager"
	"../networking"
	"../simdriver"
	"encoding/json"
	"log"
)

type OrderAssignment struct {
	Button   simdriver.ClickEvent
	Assignee networking.IP
}

type Event struct {
	Type     EventType
	Button   simdriver.ClickEvent //Etasje og opp/ned
	NewState manager.State
}

type EventType int

const (
	//Click events - clickEvent (floor og btnType)
	NewExternalOrder EventType = iota
	NewInternalOrder
	SelfAssignedOrder

	//Sensor events - event type, floor
	PassingFloor
	DoorOpenedByInternalOrder
	StoppingToFinishOrder
	LeavingFloor

	//Only state needed
	SendState
)

type LiftEvents struct {
	FloorReached chan int
	StopButton   chan bool
	Obstruction  chan bool
}
