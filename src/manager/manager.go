package manager

import (
	"fmt"
	"../simdriver"
	"time"
)

const(
	stopTime float32 = 3
	floorTravelTime float32 = 2
	floorPassingTime float32 = 0.5
)

var upQueue [4]bool
var downQueue [4]bool
var internalQueue [4]bool

func getElevWaitTime(targetFloor int, direction int) float32{
	wtime := 0

	/*
	Given that the order is assigned to this elevator
	- Find the time until the order is finished by simulating the elevator behaviour

	



	if internalQueue[last_passed_floor] || isOrder(last_passed_floor, current_direction){
    			simdriver.SetMotorDirection(simdriver.MotorStop)
    			simdriver.SetDoorOpenLamp(true)
    			deadline_timer.Stop()
    			door_timer.Reset(door_period)
    			state = doorOpen
    		} else if isOrderAhead(last_passed_floor, current_direction){ //Ordre framover
    			switch(current_direction){
    			case Up:
    				simdriver.SetMotorDirection(simdriver.MotorUp) //Kanskje bare gjøre dette hvis det endrer noe?
    			case Down:
    				simdriver.SetMotorDirection(simdriver.MotorDown)
    			}
    			deadline_timer.Reset(deadline_period)
    			state = movingBetween

    		} else if isOrderBehind(last_passed_floor, current_direction) || isOrder(last_passed_floor, oppositeDirection(current_direction)){ //Ordre bakover
    			current_direction = oppositeDirection(current_direction)
    		} 



	
	+ (traveltime - travelTimer) + (doortime - doorTimer)
	*/
}
