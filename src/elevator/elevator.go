package elevator

import (
    "time"
    "../simdriver"
)


const deadline_period = 5 * time.Second
const door_period = 3 * time.Second

type FloorOrders map[int]bool
type Orders map[simdriver.BtnType]FloorOrders

type state_t int
const (
    atFloor state_t = iota
    doorOpen
    movingBetween
)

type Direction_t int
const (
	Up Direction_t = iota
	Down
)

type request_t int
const (
	isOrderAhead = iota
	isOrderBehind
)

type readDirection struct {
    floor int
    direction Direction_t
    request request_t
    resp chan bool
}
type readOrder struct {
    order simdriver.ClickEvent
    resp chan bool
}
type deleteOp struct {
    floor int
    resp chan bool
}

func Run(
	completed_floor	chan<- int,
	missed_deadline	chan<- bool,
	floor_reached	<-chan  int,
	new_order 		<-chan  simdriver.ClickEvent,
	new_direction	chan<- Direction_t){

	readDirs := make(chan readDirection)
	readOrders := make(chan readOrder)
    deletes := make(chan deleteOp)

    reply_chan := make(chan bool)

	go func(){
		orders := make(Orders)
		orders[simdriver.Up] = make(FloorOrders)
		orders[simdriver.Down] = make(FloorOrders)
		orders[simdriver.Command] = make(FloorOrders)
		orders.Init()
		for{
			select {
			case order := <- new_order:
				orders.addOrder(order)
            case readDir := <-readDirs:
            	switch(readDir.request){
            	case isOrderAhead:
            		readDir.resp <- orders.isOrderAhead(readDir.floor, readDir.direction)
            	case isOrderBehind:
            		readDir.resp <- orders.isOrderBehind(readDir.floor, readDir.direction)
            	}
            case readOrder := <- readOrders:
            	readOrder.resp <- orders.isOrder(readOrder.order)
            case delete := <-deletes:
                orders.clearOrders(delete.floor)
                delete.resp <- true
            }
		}
	}()

	deadline_timer := time.NewTimer(deadline_period)
    deadline_timer.Stop()

    door_timer := time.NewTimer(door_period)
    door_timer.Stop()

    state := atFloor
    current_direction := Up
    last_passed_floor := simdriver.GetCurrentFloor()
    if(last_passed_floor == -1){
    	//Even though the driver initialized the elevator to a valid floor, it seems to be something wrong
    }

    for{
    	switch(state){
    	case atFloor:
    		//Noen vil av, eller på i riktig retning
    		readOrders <- readOrder{simdriver.ClickEvent{last_passed_floor, simdriver.Command}, reply_chan}
    		internal0rderAtThisFloor := <- reply_chan
    		readOrders <- readOrder{simdriver.ClickEvent{last_passed_floor, current_direction.toBtnType()}, reply_chan}
    		orderForwardAtThisFloor := <- reply_chan

    		if internal0rderAtThisFloor || orderForwardAtThisFloor {
    			simdriver.SetMotorDirection(simdriver.MotorStop)
    			simdriver.SetDoorOpenLamp(true)
    			deadline_timer.Stop()
    			door_timer.Reset(door_period)
    			state = doorOpen
    			break
    		}

    		readDirs <- readDirection{last_passed_floor, current_direction, isOrderAhead, reply_chan}
    		orderAhead := <- reply_chan

    		if orderAhead {
    			switch(current_direction){
    			case Up:
    				simdriver.SetMotorDirection(simdriver.MotorUp) //Kanskje bare gjøre dette hvis det endrer noe?
    			case Down:
    				simdriver.SetMotorDirection(simdriver.MotorDown)
    			}
    			deadline_timer.Reset(deadline_period)
    			state = movingBetween
    			break
    		}

    		readDirs <- readDirection{last_passed_floor, current_direction, isOrderBehind, reply_chan}
    		orderBehind := <- reply_chan
    		readOrders <- readOrder{simdriver.ClickEvent{last_passed_floor, current_direction.oppositeDirection().toBtnType()}, reply_chan}
    		orderBackwardAtThisFloor := <- reply_chan

    		if orderBehind || orderBackwardAtThisFloor{ //Ordre bakover
    			current_direction = current_direction.oppositeDirection()
    		}

    	case doorOpen:
    		select{ //Trenger vel ikke select her?
    		case <- door_timer.C:
    			simdriver.SetDoorOpenLamp(false)
                state = atFloor
                //completed_floor <- last_passed_floor
                deletes <- deleteOp{last_passed_floor, reply_chan}
                <- reply_chan
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
    	time.Sleep(5 * time.Millisecond)
    }
}

func (orders Orders) isOrder(event simdriver.ClickEvent) bool {
	if event.Floor < 0 || event.Floor > simdriver.NumFloors-1 {
		//Handle invalid floor
		return false
	} else if event.Type == simdriver.Up && event.Floor == simdriver.NumFloors-1 {
		//Handle invalid button
		return false
	} else if event.Type == simdriver.Down && event.Floor == 0 {
		//Handle invalid button
		return false
	}
	return orders[event.Type][event.Floor]
}

func (orders Orders) isOrderAhead(currentFloor int, direction Direction_t) bool {
	for _, buttonOrders := range orders{
		for floor, isSet := range buttonOrders {
			if direction == Up && floor > currentFloor && isSet {
				return true
			} else if direction == Down && floor < currentFloor && isSet {
				return true
			}
		}
	}
	return false
}

func (orders Orders) isOrderBehind(currentFloor int, direction Direction_t) bool{
	return orders.isOrderAhead(currentFloor, direction.oppositeDirection())
}

func (orders Orders) addOrder(order simdriver.ClickEvent){
	switch(order.Type){
	case simdriver.Up:
		if order.Floor < 0 || order.Floor > simdriver.NumFloors-2 {
			//Handle error
			return
		}
	case simdriver.Down:
		if order.Floor < 1 || order.Floor > simdriver.NumFloors-1 {
			//Handle error
			return
		}
	case simdriver.Command:
		if order.Floor >= simdriver.NumFloors || order.Floor < 0 {
			//Handle error
			return
		}
	}
	orders[order.Type][order.Floor] = true
	simdriver.SetBtnLamp(order.Floor, order.Type, true)
}

func (direction Direction_t) oppositeDirection() Direction_t {
	if direction == Up{
		return Down
	} else {
		return Up
	}
}

func (orders Orders) clearOrders(floor int){
	if floor != 0 {
		simdriver.SetBtnLamp(floor, simdriver.Down, false)
		orders[simdriver.Down][floor] = false
	}
	if floor != (simdriver.NumFloors-1) {
		simdriver.SetBtnLamp(floor, simdriver.Up, false)
		orders[simdriver.Up][floor] = false
	}
	simdriver.SetBtnLamp(floor, simdriver.Command, false)
	orders[simdriver.Command][floor] = false
}

func (orders Orders) Init(){
	for floor := 0; floor < simdriver.NumFloors; floor++ {
		if floor == 0{
			orders[simdriver.Up][floor] = false
			orders[simdriver.Command][floor] = false
		} else if floor == simdriver.NumFloors - 1 {
			orders[simdriver.Down][floor] = false
			orders[simdriver.Command][floor] = false
		} else {
			orders[simdriver.Up][floor] = false
			orders[simdriver.Down][floor] = false
			orders[simdriver.Command][floor] = false
		}
	}
}

func (direction Direction_t) toBtnType() simdriver.BtnType {
	if direction == Up{
		return simdriver.Up
	} else {
		return simdriver.Down
	}
}
