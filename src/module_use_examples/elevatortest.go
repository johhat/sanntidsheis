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

	simdriver.Init(clickEvent_chan, sensorEvent_chan)
	time.Sleep(5 * time.Millisecond)
	go elevator.Run(completed_floor_chan, missed_deadline_chan, floor_reached_chan)
	go func(s_event, floor_reached chan int){
		for{
			floor := <- s_event
			if floor != -1{
				floor_reached <- floor
			}
		}
	}(sensorEvent_chan, floor_reached_chan)
	go func(cl_event chan simdriver.ClickEvent){
		for{
			event := <- cl_event
			if event.Type == simdriver.Up{
				elevator.AddOrderExternal(event.Floor, elevator.Up)
			} else if event.Type == simdriver.Down{
				elevator.AddOrderExternal(event.Floor, elevator.Down)
			} else {
				elevator.AddOrderInternal(event.Floor)
			}
		}
	}(clickEvent_chan)

	/*time.Sleep(10 * time.Millisecond)

	elevator.AddOrderExternal(0, elevator.Up)
	elevator.AddOrderExternal(1, elevator.Up)
	elevator.AddOrderExternal(1, elevator.Down)
	elevator.AddOrderExternal(2, elevator.Up)
	elevator.AddOrderExternal(2, elevator.Down)
	elevator.AddOrderExternal(3, elevator.Down)
	elevator.AddOrderInternal(0)
	elevator.AddOrderInternal(1)
	elevator.AddOrderInternal(2)
	elevator.AddOrderInternal(3)*/
	for{
		time.Sleep(10 * time.Millisecond)

	}

	
}
