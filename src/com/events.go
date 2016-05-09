package com

type EventType int

const (
	PassingFloor EventType = iota
	StoppingToFinishOrder
	LeavingFloor
	DoorClosed
	DirectionChanged
)
