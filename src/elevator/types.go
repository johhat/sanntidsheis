package elevator

import (
	driver "../driver"
)

type FloorOrders map[int]bool
type Orders map[driver.BtnType]FloorOrders

type ReadDirection struct {
	Floor     int
	Direction Direction_t
	Request   request_t
	Resp      chan bool
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
)

type request_t int

const (
	IsOrderAhead request_t = iota
	IsOrderBehind
)

type Direction_t int

const (
	Up Direction_t = iota
	Down
)

func (direction Direction_t) OppositeDirection() Direction_t {
	if direction == Up {
		return Down
	} else {
		return Up
	}
}

func (direction Direction_t) toBtnType() driver.BtnType {
	if direction == Up {
		return driver.Up
	} else {
		return driver.Down
	}
}
