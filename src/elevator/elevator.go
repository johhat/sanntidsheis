package elevator

import (
    "time"
    "../simdriver"
    //"fmt"
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
	Up direction_t = iota
	Down
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
    current_direction := Up
    last_passed_floor = simdriver.GetCurrentFloor()
    if(last_passed_floor == -1){
    	//Even though the driver initialized the elevator to a valid floor, it seems to be something wrong
    }
    for{
    	switch(state){
    	case atFloor:
    		//Noen vil av, eller på i riktig retning
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
    	case doorOpen:
    		select{ //Trenger vel ikke select her?
    		case <- door_timer.C:
    			simdriver.SetDoorOpenLamp(false)
                state = atFloor
                //completed_floor <- last_passed_floor
                clearOrders(last_passed_floor)
    		}
    	case movingBetween:
    		select{
    			case floor := <- floor_reached:
    				last_passed_floor = floor
    				simdriver.SetFloorIndicator(floor)
    				state = atFloor
					deadline_timer.Stop()
    			case <- deadline_timer.C:
    				missed_deadline <- true
    		}
    	}
    	time.Sleep(50 * time.Millisecond)
    }
}

func isOrderAhead(currentFloor int, direction direction_t) bool{
	if(direction == Up){
		ordersAbove := []bool{}
		ordersAbove = append(ordersAbove, upQueue[currentFloor+1:]...)
		ordersAbove = append(ordersAbove, downQueue[currentFloor+1:]...)
		ordersAbove = append(ordersAbove,internalQueue[currentFloor+1:]...)
		for _, v := range ordersAbove{
			if v {
				return true
			}
		}
		return false
	} else {
		ordersBelow := []bool{}
		ordersBelow = append(ordersBelow, upQueue[:currentFloor]...)
		ordersBelow = append(ordersBelow, downQueue[:currentFloor]...)
		ordersBelow = append(ordersBelow, internalQueue[:currentFloor]...)
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
	if(direction == Down){
		ordersAbove := []bool{}
		ordersAbove = append(ordersAbove, upQueue[currentFloor+1:]...)
		ordersAbove = append(ordersAbove, downQueue[currentFloor+1:]...)
		ordersAbove = append(ordersAbove,internalQueue[currentFloor+1:]...)
		for _, v := range ordersAbove{
			if v {
				return true
			}
		}
		
		return false
	} else {
		ordersBelow := []bool{}
		ordersBelow = append(ordersBelow, upQueue[:currentFloor]...)
		ordersBelow = append(ordersBelow, downQueue[:currentFloor]...)
		ordersBelow = append(ordersBelow, internalQueue[:currentFloor]...)
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
	simdriver.SetBtnLamp(floor, simdriver.Command, true)
}

func AddOrderExternal(floor int, direction direction_t){
	switch(direction){
	case Up:
		if floor < 0 || floor > simdriver.NumFloors-2 {
			// Handle invalid order
			return
		} else {
			upQueue[floor] = true
			simdriver.SetBtnLamp(floor, simdriver.Up, true)
		}
	case Down:
		if floor < 1 || floor > simdriver.NumFloors-1 {
			// Handle invalid order
			return
		} else {
			downQueue[floor] = true
			simdriver.SetBtnLamp(floor, simdriver.Down, true)
		}
	}
}

func oppositeDirection(direction direction_t) direction_t {
	if direction == Up{
		return Down
	} else {
		return Up
	}
}

func isOrder(floor int, direction direction_t) bool {
	if direction == Up{
		if floor < 0 || floor > simdriver.NumFloors-2 {
			// Handle invalid input
			return false
		} else {
			return upQueue[floor]
		}
	} else {
		if floor < 1 || floor > simdriver.NumFloors-1 {
			// Handle invalid input
			return false
		} else {
			return downQueue[floor]
		}
	}
}

func clearOrders(floor int){
	upQueue[floor] = false
	downQueue[floor] = false
	internalQueue[floor] = false
	if floor != 0 {
		simdriver.SetBtnLamp(floor, simdriver.Down, false)
	}
	if floor != (simdriver.NumFloors-1) {
		simdriver.SetBtnLamp(floor, simdriver.Up, false)
	}
	simdriver.SetBtnLamp(floor, simdriver.Command, false)
}

