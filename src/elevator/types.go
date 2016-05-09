package elevator

import (
	driver "../driver"
)

type FloorOrders map[int]bool
type Orders map[driver.BtnType]FloorOrders

type ReadDirection struct {
	Floor   int
	Dir     Direction
	Request request_t
	Resp    chan bool
}

type ReadOrder struct {
	Order driver.ClickEvent
	Resp  chan bool
}

type state_t int

const (
	atFloor state_t = iota
	doorOpen
	movingBetween
	errorState
	reInitState
)

type request_t int

const (
	IsOrderAhead request_t = iota
	IsOrderBehind
)

type Direction int

const (
	Up Direction = iota
	Down
)

func (direction Direction) OppositeDirection() Direction {
	if direction == Up {
		return Down
	} else {
		return Up
	}
}

func (direction Direction) toBtnType() driver.BtnType {
	if direction == Up {
		return driver.Up
	} else {
		return driver.Down
	}
}
