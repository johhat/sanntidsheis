package main

// Package for testing the simulator driver/TCP client

import (
	"../simdriver"
	"fmt"
	"runtime"
)

func main() {
	//runtime.GOMAXPROCS(runtime.NumCPU())
	var exit_chan = make(chan int)
	var clickEvent_chan = make(chan simdriver.ClickEvent)
	var sensorEvent_chan = make(chan int)
	simdriver.Init(clickEvent_chan, sensorEvent_chan)
	go func() {
		for {
			select {
			case c_event := <-clickEvent_chan:
				fmt.Println("Clickevent", c_event.String())
			case s_event := <-sensorEvent_chan:
				fmt.Println("Sensorevent", s_event)
			}
		}
	}()
	simdriver.SetBtnLamp(2, simdriver.Up, true)
	simdriver.SetBtnLamp(0, simdriver.Command, true)
	simdriver.SetBtnLamp(1, simdriver.Down, true)

	<-exit_chan
}
