package manager

import (
	"fmt"
	"../simdriver"
	"time"
	"../elevator"
)

var active bool

//Local orders
//Remote orders 
//Remote states
//Local state

const(
	stopTime float32 = 3
	floorTravelTime float32 = 2
	floorPassingTime float32 = 0.5
	movingPenalty float32 = floorTravelTime/2
	doorOpenPenalty float32 = doorOpenTime/2
)

//Pseudocode, not runnable

//Argument mot å bruke dette som kostnadsfunksjon: Alle heiser må vite om de andres interne ordre
func getExpectedResponseTime(targetFloor int, direction direction_t, id networking.ID) (responseTime, bestCaseTime, worstCaseTime float32){
	if (direction == Up && floor == NumFloors-1) || (direction == Down && floor == 0){
		//Handle error
	}
	/*
	Given that the order is assigned to this elevator
	- Find the time until the order is cleared by simulating the elevator behaviour
	- Then find best case and worst case time until the passenger will get to the 
	  destination floor, based on the other orders in the queue and the possible
	  destinations (max 3, min 1, average 2)
	*/

	//Among the possible destinations, find the floors closest and furthest away
	var bestCaseFloor int
	var worstCaseFloor int
	if direction == Up {
		worstCaseFloor = NumFloors-1
		bestCaseFloor = currentFloor+1
	} else {
		worstCaseFloor = 0
		bestCaseFloor = currentFloor-1
	}
	
	//Initialize variables
	responseTime := 0
	bestCaseTime := 0
	worstCaseTime := 0

	currentOrders := orders[id]
	orders[id].append(targetFloor, direction)
	currentDirection := direction[id]
	currentFloor := elevator[id]

	if targetFloor == currentFloor && direction == currentDirection{
		responseTime = 0
	}

	for{
		if internalQueue[last_passed_floor] || isOrder(last_passed_floor, current_direction){
			//Clear orders on floor
			if !orderCleared{
				wtime += stopTime
			}
    	} else if isOrderAhead(last_passed_floor, current_direction){ //Ordre framover
    		wtime += floorTravelTime
    		currentFloor = nextFloor
    	} else if isOrderBehind(last_passed_floor, current_direction) || isOrder(last_passed_floor, oppositeDirection(current_direction)){ //Ordre bakover
    		current_direction = oppositeDirection(current_direction)
    	} else {
    		//No orders left, to prevent erronous infinite loop this must be catched
    	}
	}

	if bestCaseTime == 0 || worstCaseTime == 0{
		//Handle error
	}
)


func Run(

	) {
	//Anything to do at the beginning, or maybe create an init function?
	for {
		select {
			//case elevatorDisconnected:
				//Bestem hvilke av ordrene denne heisen har som skal tas lokalt
			//case elevatorAdded:
				//send vår tilstand til den nye heisen
			//case buttonClicked:
				//Hvis dette er en ny ordre
					//Hvis intern ordre
						//Legg til ordre lokalt, gi beskjed til de andre heisene
					//Hvis ekstern ordre
						//Sjekk forventet responstid for alle heiser, velg den beste
				//Hvis stoppknapp
					//Hvis ikke aktiv -> bli aktiv
					//Init på nytt?
			//case sensorEvent:
			//case assignedOrder:
				//legg til ordre lokalt
			//case localOrderFinished:
			//case localTimeout:
				// active = false
				// disconnect TCP eller varsle de andre heisene
			//case remoteOrderFinished:
				//Sjekk om mottatt ordreliste samsvarer med gammel-fullført ordre
				//Oppdater oversikt over den andre heisens ordre
		}
	}
}

func saveInternalOrders(){
	//Skriv alle interne ordre til harddisk
	//Hvordan sikre at filen aldri blir korrupt hvis vi avbrytes under skriving?
}

func restoreInternalOrders(){
	//Les lagrede interne ordre fra disk hvis de finnes og ikke er korrupte
}
