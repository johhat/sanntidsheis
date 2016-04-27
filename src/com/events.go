package com

type EventType int

const (
	//Order events
	NewExternalOrder EventType = iota
	NewInternalOrder

	//Sensor events
	PassingFloor
	DoorOpenedByInternalOrder
	StoppingToFinishOrder
	LeavingFloor
	DoorClosed
	DirectionChanged
)
