package manager

import (
	"fmt"
	"../simdriver"
	"time"
	"../elevator"
)

const(
	stopTime float32 = 3
	floorTravelTime float32 = 2
	floorPassingTime float32 = 0.5
)

//Pseudocode, not runnable
func getElevWaitTime(targetFloor int, direction int) float32{
	wtime := 0

	currentOrders := elevator.getOrders()
	//Add the new order to the order list
	currentDirection := elevator.getDirection()
	currentFloor := elevator.getCurrentFloor()

	if targetFloor == currentFloor && direction == currentDirection{
		return 0
	}
	if no

	for{
		if internalQueue[last_passed_floor] || isOrder(last_passed_floor, current_direction){
			//Clear orders on floor
			if orderCleared{
				return wtime
			}
    		wtime += stopTime
    	} else if isOrderAhead(last_passed_floor, current_direction){ //Ordre framover
    		wtime += floorTravelTime
    		currentFloor = nextFloor
    	} else if isOrderBehind(last_passed_floor, current_direction) || isOrder(last_passed_floor, oppositeDirection(current_direction)){ //Ordre bakover
    		current_direction = oppositeDirection(current_direction)
    	} 
	}

	/*
	Given that the order is assigned to this elevator
	- Find the time until the order is cleared by simulating the elevator behaviour
	- Then find best case and worst case time until the passenger will get to the 
	  destination floor, based on the other orders in the queue

	
	


	



	
	+ (traveltime - travelTimer) + (doortime - doorTimer)
	*/

)

/*
Events:
Nettverk mistet heis(er)
Nettverk fikk kontakt med en heis
Timeout for heis
Lokal heis har fullført etasje
Ny ordre gis til lokal heis
Knapp trykkes på lokal heis
Hy ordre fra nettverk / knappetrykk fra nettverk

*/
func Run() {
	for {
		select {}
	}
}
