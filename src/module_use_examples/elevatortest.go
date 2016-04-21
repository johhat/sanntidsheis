package main

import (
	"../elevator"
	"../simdriver"
	//"fmt"
	"time"
	//"runtime"
)

func main() {
	//runtime.GOMAXPROCS(runtime.NumCPU())
	var sensorEvent_chan = make(chan int)
	var clickEvent_chan = make(chan simdriver.ClickEvent)

	var completed_floor_chan = make(chan int)
	var missed_deadline_chan = make(chan bool)
	var floor_reached_chan = make(chan int)
	//var new_order_chan = make(chan simdriver.ClickEvent)
	var new_direction_chan = make(chan elevator.Direction_t)

	simdriver.Init(clickEvent_chan, sensorEvent_chan)
	time.Sleep(5 * time.Millisecond)
	go elevator.Run(completed_floor_chan, missed_deadline_chan, floor_reached_chan, clickEvent_chan, new_direction_chan)
	go func(s_event, floor_reached chan int){
		for{
			floor := <- s_event
			if floor != -1{
				floor_reached <- floor
			}
		}
	}(sensorEvent_chan, floor_reached_chan)

	for{
		time.Sleep(10 * time.Millisecond)

	}

	
}
