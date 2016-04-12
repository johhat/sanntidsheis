package elevator

import (
    "time"
    "../simdriver"
)


const deadline_period = 5 * time.Second //Juster denne
const door_period = 3 * time.Second
var last_passed_floor int

//Midlertidig usikker kø med fast størrelse
var upQueue [simdriver.NumFloors]bool
var downQueue [simdriver.NumFloors]bool
var internalQueue [simdriver.NumFloors]bool

type state_t int
const (
    atFloor state_t = iota
    doorOpen
    movingBetween
)

type direction_t int
const (
	up direction_t = iota
	down
)

func Run(
	completed_floor  chan <- int,
	missed_deadline  chan <- bool,
	floor_reached    <- chan int){

	deadline_timer := time.NewTimer(deadline_period)
    deadline_timer.Stop()

    door_timer := time.NewTimer(door_period)
    door_timer.Stop()

    state := atFloor
    current_direction := up
    last_passed_floor = simdriver.getFloorSensorSignal()
    if(last_passed_floor == -1){
    	//Even though the driver initialized the elevator to a valid floor, it seems to be something wrong
    }

    for{
    	switch(state){
    	case atFloor:
    		//Noen vil av, eller på i riktig retning
    		if internalQueue[last_passed_floor] || isOrder(last_passed_floor, current_direction){
    			simdriver.setMotorDirection(MotorStop)
    			simdriver.setDoorOpenLamp(true)
    			deadline_timer.Stop()
    			door_timer.Reset(door_period)
    			state = doorOpen
    		}
    		//Ordre framover
    		else if isOrderAhead(last_passed_floor, current_direction){
    			switch(current_direction){
    			case up:
    				simdriver.setMotorDirection(MotorUp) //Kanskje bare gjøre dette hvis det endrer noe?
    			case down:
    				simdriver.setMotorDirection(MotorDown)
    			}
    			deadline_timer.Reset(deadline_period)
    			state = movingBetween
    		}
    		//Ordre bakover
    		else if isOrderBehind(last_passed_floor, current_direction) || isOrder(last_passed_floor, oppositeDirection(current_direction)){
    			current_direction = oppositeDirection(current_direction)
    		}
    	case doorOpen:
    		select{ //Trenger vel ikke select her?
    		case <- door_timer.C:
    			simdriver.setDoorOpenLamp(false)
                state = atFloor
                completed_floor <- last_passed_floor
                clearOrders(last_passed_floor)
    		}
    	case movingBetween:
    		select{
    			case floor := <- floor_reached:
    				last_passed_floor = floor
    				simdriver.setFloorIndicator(floor)
    				state = atFloor
					deadline_timer.Stop()
    			case <- deadline_timer.C:
    				missed_deadline <- true
    		}
    	}
    }
}

func isOrderAhead(currentFloor int, direction direction_t) bool{
	if(direction == up){
		ordersAbove = append(upQueue[currentFloor+1:],downQueue[currentFloor+1:],internalQueue[currentFloor+1:]...)
		for _, v := range ordersAbove{
			if v {
				return true
			}
		}
		return false
	}
	else{
		ordersBelow = append(upQueue[:currentFloor],downQueue[:currentFloor],internalQueue[:currentFloor]...)
		for _, v := range ordersBelow{
			if v {
				return true
			}
		}
		return false
	}
}

func isOrderBehind(currentFloor int, direction direction_t) bool{
	//Sjekk etter ugydig etasje
	if(direction == down){
		ordersAbove = append(upQueue[currentFloor+1:],downQueue[currentFloor+1:],internalQueue[currentFloor+1:]...)
		for _, v := range ordersAbove{
			if v {
				return true
			}
		}
		return false
	}
	else{
		ordersBelow = append(upQueue[:currentFloor],downQueue[:currentFloor],internalQueue[:currentFloor]...)
		for _, v := range ordersBelow{
			if v {
				return true
			}
		}
		return false
	}
}

func AddOrderInternal(floor int){
	if floor >= simdriver.NumFloors || floor < 0 {
		//Handle invalid order
		return
	}

	internalQueue[floor] = true
}

func AddOrderExternal(floor int, direction direction_t){
	switch(direction){
	case up:
		if floor < 0 || floor > simdriver.NumFloors-2 {
			// Handle invalid order
			return
		}
		else{
			upQueue[floor] = true
		}
	case down:
		if floor < 1 || floor > simdriver.NumFloors-1 {
			// Handle invalid order
			return
		}
		else{
			downQueue[floor] = true
		}
	}
}

func oppositeDirection(direction direction_t) direction_t {
	switch (direction){
	case up:
		return down
	case down:
		return up
	}
}

func isOrder(floor int, direction direction_t) bool {
	switch(direction){
	case up:
		if floor < 0 || floor > simdriver.NumFloors-2 {
			// Handle invalid input
			return false
		}
		else{
			return upQueue[floor]
		}
	case down:
		if floor < 1 || floor > simdriver.NumFloors-1 {
			// Handle invalid input
			return false
		}
		else{
			return downQueue[floor]
		}
	}
}

func clearOrders(floor int){
	upQueue[floor] = false
	downQueue[floor] = false
	internalQueue[floor] = false
}

