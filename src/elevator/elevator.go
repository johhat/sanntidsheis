package elevator

import (
	"../simdriver"
	"time"
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
	floor     int
	direction Direction_t
	request   request_t
	resp      chan bool
}
type readOrder struct {
	order simdriver.ClickEvent
	resp  chan bool
}
type deleteOp struct {
	floor int
	resp  chan bool
}

func Run(
	completed_floor 	chan<- int,
	elev_error		 	chan<- bool,
	floor_reached 		<-chan int,
	new_order 			<-chan simdriver.ClickEvent,
	new_direction 		chan<- Direction_t,
	door_closed_chan	chan<- bool,
	readDirs			chan<- readDirection,
	readOrders			chan<- readOrder,
	start_moving		chan<- bool,
	passingFloor_chan	chan<- bool,
	deletes				chan<- deleteOp) {

	reply_chan := make(chan bool)

	/*go func() {
		orders := make(Orders)
		orders[simdriver.Up] = make(FloorOrders)
		orders[simdriver.Down] = make(FloorOrders)
		orders[simdriver.Command] = make(FloorOrders)
		orders.Init()
		for {
			select {
			case order := <-new_order:
				orders.addOrder(order)
			case readDir := <-readDirs:
				switch readDir.request {
				case isOrderAhead:
					readDir.resp <- orders.isOrderAhead(readDir.floor, readDir.direction)
				case isOrderBehind:
					readDir.resp <- orders.isOrderBehind(readDir.floor, readDir.direction)
				}
			case readOrder := <-readOrders:
				readOrder.resp <- orders.isOrder(readOrder.order)
			case delete := <-deletes:
				orders.clearOrders(delete.floor)
				delete.resp <- true
			}
		}
	}()*/

	deadline_timer := time.NewTimer(deadline_period)
	deadline_timer.Stop()

	door_timer := time.NewTimer(door_period)
	door_timer.Stop()

	state := atFloor
	current_direction := Up
	last_passed_floor := simdriver.GetCurrentFloor()
	if last_passed_floor == -1 {
		//Even though the driver initialized the elevator to a valid floor, it seems to be something wrong
		//Run init again or crash the program?
	}

	passingFloor := true

	for {
		switch state {
		case atFloor:
			//Noen vil av, eller på i riktig retning
			readOrders <- readOrder{simdriver.ClickEvent{last_passed_floor, simdriver.Command}, reply_chan}
			internal0rderAtThisFloor := <-reply_chan
			readOrders <- readOrder{simdriver.ClickEvent{last_passed_floor, current_direction.toBtnType()}, reply_chan}
			orderForwardAtThisFloor := <-reply_chan

			if internal0rderAtThisFloor || orderForwardAtThisFloor {
				passingFloor = false
				simdriver.SetMotorDirection(simdriver.MotorStop)
				simdriver.SetDoorOpenLamp(true)
				completed_floor <- last_passed_floor
				deadline_timer.Stop()
				door_timer.Reset(door_period)
				state = doorOpen
				break
			}

			readDirs <- readDirection{last_passed_floor, current_direction, isOrderAhead, reply_chan}
			orderAhead := <-reply_chan

			if orderAhead {
				start_moving <- true
				if(passingFloor){
					passingFloor_chan <- true
				}
				switch current_direction {
				case Up:
					simdriver.SetMotorDirection(simdriver.MotorUp)
				case Down:
					simdriver.SetMotorDirection(simdriver.MotorDown)
				}
				deadline_timer.Reset(deadline_period)
				state = movingBetween
				passingFloor = true
				break
			}

			readDirs <- readDirection{last_passed_floor, current_direction, isOrderBehind, reply_chan}
			orderBehind := <-reply_chan
			readOrders <- readOrder{simdriver.ClickEvent{last_passed_floor, current_direction.OppositeDirection().toBtnType()}, reply_chan}
			orderBackwardAtThisFloor := <-reply_chan

			if orderBehind || orderBackwardAtThisFloor {
				current_direction = current_direction.OppositeDirection()
			}

		case doorOpen:
			<-door_timer.C:
			simdriver.SetDoorOpenLamp(false)
			state = atFloor
			door_closed_chan <- true
			deletes <- deleteOp{last_passed_floor, reply_chan}
			<-reply_chan
		case movingBetween:
			select {
			case floor := <-floor_reached:
				last_passed_floor = floor
				simdriver.SetFloorIndicator(floor)
				state = atFloor
				deadline_timer.Stop()
			case <-deadline_timer.C:
				elev_error <- true
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (direction Direction_t) OppositeDirection() Direction_t {
	if direction == Up {
		return Down
	} else {
		return Up
	}
}

func (direction Direction_t) toBtnType() simdriver.BtnType {
	if direction == Up {
		return simdriver.Up
	} else {
		return simdriver.Down
	}
}
