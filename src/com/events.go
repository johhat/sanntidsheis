package com

type EventType int

const (
	PassingFloor EventType = iota
	DoorOpenedByInternalOrder
	StoppingToFinishOrder
	LeavingFloor
	DoorClosed
	DirectionChanged
)
