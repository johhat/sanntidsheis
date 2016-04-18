package com

import (
	"../networking"
	"../simdriver"
	"encoding/json"
)

type Order struct {
	Button  simdriver.OrderButton
	TakenBy network.ID
	Done    bool
}

type LiftEvents struct {
	FloorReached chan int
	StopButton   chan bool
	Obstruction  chan bool
}
