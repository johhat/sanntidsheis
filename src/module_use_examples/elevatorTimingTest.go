package main

// This is not directly part of the project, but is used to test the
// travel speed of the elevators at the lab

import (
	"../simdriver"
	"fmt"
	"time"
)

func main() {

	var exit_chan = make(chan int)
	var clickEvent_chan = make(chan simdriver.ClickEvent)
	var sensorEvent_chan = make(chan int)

	var completed_floor_chan = make(chan int)
	var missed_deadline_chan = make(chan bool)
	var floor_reached_chan = make(chan int)

	simdriver.Init(clickEvent_chan, sensorEvent_chan)
	time.Sleep(5 * time.Millisecond)
	go elevator.Run(completed_floor_chan, missed_deadline_chan, floor_reached_chan)
	go func(s_event, floor_reached chan int) {
		for {
			floor := <-s_event
			fmt.Println("Sensor event", floor, "at time", time.Now())
			if floor != -1 {
				floor_reached <- floor
			}
		}
	}(sensorEvent_chan, floor_reached_chan)

	elevator.AddOrderInternal(0)
	time.Sleep(10 * time.Second)
	elevator.AddOrderInternal(3)

	<-exit_chan
}
