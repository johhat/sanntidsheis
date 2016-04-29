package elevator

import (
	driver "../driver"
	"fmt"
	"time"
)

const (
	deadline_period = 5 * time.Second
	door_period     = 3 * time.Second
)

var current_direction Direction_t = Up

func GetCurrentDirection() Direction_t {
	return current_direction
}

func Run(
	completed_floor chan<- int,
	elev_error chan<- bool,
	floor_reached <-chan int,
	new_order <-chan driver.ClickEvent,
	new_direction chan<- Direction_t,
	door_closed_chan chan<- bool,
	readDirs chan<- ReadDirection,
	ReadOrders chan<- ReadOrder,
	start_moving chan<- bool,
	passingFloor_chan chan<- bool) {

	reply_chan := make(chan bool)

	deadline_timer := time.NewTimer(deadline_period)
	deadline_timer.Stop()

	door_timer := time.NewTimer(door_period)
	door_timer.Stop()

	state := atFloor
	last_passed_floor := driver.GetFloorSensorSignal()
	if last_passed_floor == -1 {
		//Even though the driver initialized the elevator to a valid floor, it seems to be something wrong
		//Run init again or crash the program?
	}

	passingFloor := false

	for {
		switch state {
		case atFloor:
			//Noen vil av, eller på i riktig retning
			ReadOrders <- ReadOrder{driver.ClickEvent{last_passed_floor, driver.Command}, reply_chan}
			internal0rderAtThisFloor := <-reply_chan
			ReadOrders <- ReadOrder{driver.ClickEvent{last_passed_floor, current_direction.toBtnType()}, reply_chan}
			orderForwardAtThisFloor := <-reply_chan

			if internal0rderAtThisFloor || orderForwardAtThisFloor {
				fmt.Println("\033[31m" + "Elevator: stopping" + "\033[0m")
				passingFloor = false
				driver.SetMotorDirection(driver.MotorStop)
				driver.SetDoorOpenLamp(true)
				completed_floor <- last_passed_floor
				deadline_timer.Stop()
				door_timer.Reset(door_period)
				state = doorOpen
				break
			}

			readDirs <- ReadDirection{last_passed_floor, current_direction, IsOrderAhead, reply_chan}
			orderAhead := <-reply_chan

			if orderAhead {
				start_moving <- true
				if passingFloor {
					passingFloor_chan <- true
				}
				switch current_direction {
				case Up:
					driver.SetMotorDirection(driver.MotorUp)
					fmt.Println("\033[31m" + "Elevator: moving up" + "\033[0m")
				case Down:
					driver.SetMotorDirection(driver.MotorDown)
					fmt.Println("\033[31m" + "Elevator: moving down" + "\033[0m")
				}
				deadline_timer.Reset(deadline_period)
				state = movingBetween
				passingFloor = true
				break
			}

			readDirs <- ReadDirection{last_passed_floor, current_direction, IsOrderBehind, reply_chan}
			orderBehind := <-reply_chan
			ReadOrders <- ReadOrder{driver.ClickEvent{last_passed_floor, current_direction.OppositeDirection().toBtnType()}, reply_chan}
			orderBackwardAtThisFloor := <-reply_chan

			if orderBehind || orderBackwardAtThisFloor {
				fmt.Println("\033[31m" + "Elevator: Changing direction" + "\033[0m")
				current_direction = current_direction.OppositeDirection()
			}

		case doorOpen:
			<-door_timer.C
			driver.SetDoorOpenLamp(false)
			state = atFloor
			door_closed_chan <- true
		case movingBetween:
			select {
			case floor := <-floor_reached:
				last_passed_floor = floor
				driver.SetFloorIndicator(floor)
				state = atFloor
				deadline_timer.Stop()
			case <-deadline_timer.C:
				elev_error <- true
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
}
