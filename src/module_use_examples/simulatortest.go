package main

// Package for testing the simulator driver/TCP client

import (
	"../simdriver"
	"fmt"
)

func main() {

	var exit_chan = make(chan int)
	var clickEvent_chan = make(chan simdriver.ClickEvent)
	var sensorEvent_chan = make(chan int)
	simdriver.Init(clickEvent_chan, sensorEvent_chan)
	for {
		select {
		case c_event := <-clickEvent_chan:
			fmt.Println("Clickevent",c_event.String())
		case s_event := <-sensorEvent_chan:
			fmt.Println("Sensorevent", s_event)
		}
	}
	<-exit_chan
}
