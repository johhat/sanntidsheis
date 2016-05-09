package elevator

import (
	driver "../driver"
	"fmt"
	"log"
	"time"
)

const (
	deadlinePeriod = 3 * time.Second
	doorPeriod     = 3 * time.Second
)

var current_direction Direction = Up

func GetCurrentDirection() Direction {
	return current_direction
}

func Run(
	completedFloor chan<- int,
	floorReached <-chan int,
	newDirection chan<- Direction,
	doorClosed chan<- bool,
	startedMoving chan<- bool,
	passingFloor chan<- bool,
	elevatorError chan<- bool,
	resumeAfterError <-chan bool,
	externalError <-chan bool,
	readDirection chan<- ReadDirection,
	readOrders chan<- ReadOrder) {

	readResult := make(chan bool)

	deadlineTimer := time.NewTimer(deadlinePeriod)
	deadlineTimer.Stop()

	doorTimer := time.NewTimer(doorPeriod)
	doorTimer.Stop()

	state := atFloor
	lastPassedFloor := driver.GetFloorSensorSignal()
	if lastPassedFloor == -1 {
		log.Fatal("[FATAL]\tElevator initializing between floors")
	}

	isPassingFloor := false

	for {
		select {
		case <-externalError:
			fmt.Println("\033[31m" + "Elevator: received external error" + "\033[0m")
			driver.SetMotorDirection(driver.MotorStop)
			elevatorError <- true
			state = errorState
		default:
		}
		switch state {
		case atFloor:
			readOrders <- ReadOrder{driver.ClickEvent{lastPassedFloor, driver.Command}, readResult}
			internal0rderAtThisFloor := <-readResult
			readOrders <- ReadOrder{driver.ClickEvent{lastPassedFloor, current_direction.toBtnType()}, readResult}
			orderForwardAtThisFloor := <-readResult

			if internal0rderAtThisFloor || orderForwardAtThisFloor {
				fmt.Println("\033[31m" + "Elevator: stopping" + "\033[0m")
				isPassingFloor = false
				driver.SetMotorDirection(driver.MotorStop)
				driver.SetDoorOpenLamp(true)
				completedFloor <- lastPassedFloor
				deadlineTimer.Stop()
				doorTimer.Reset(doorPeriod)
				state = doorOpen
				break
			}

			readDirection <- ReadDirection{lastPassedFloor, current_direction, IsOrderAhead, readResult}
			orderAhead := <-readResult

			if orderAhead {
				startedMoving <- true
				if isPassingFloor {
					passingFloor <- true
				}
				switch current_direction {
				case Up:
					driver.SetMotorDirection(driver.MotorUp)
					fmt.Println("\033[31m" + "Elevator: moving up" + "\033[0m")
				case Down:
					driver.SetMotorDirection(driver.MotorDown)
					fmt.Println("\033[31m" + "Elevator: moving down" + "\033[0m")
				}
				deadlineTimer.Reset(deadlinePeriod)
				state = movingBetween
				isPassingFloor = true
				break

			}

			readDirection <- ReadDirection{lastPassedFloor, current_direction, IsOrderBehind, readResult}
			orderBehind := <-readResult
			readOrders <- ReadOrder{driver.ClickEvent{lastPassedFloor, current_direction.OppositeDirection().toBtnType()}, readResult}
			orderBackwardAtThisFloor := <-readResult

			if orderBehind || orderBackwardAtThisFloor {
				fmt.Println("\033[31m" + "Elevator: Changing direction" + "\033[0m")
				current_direction = current_direction.OppositeDirection()
				newDirection <- current_direction
			}

		case doorOpen:
			<-doorTimer.C
			driver.SetDoorOpenLamp(false)
			state = atFloor
			doorClosed <- true
		case movingBetween:
			select {
			case floor := <-floorReached:
				if ((current_direction == Up) && (floor != lastPassedFloor+1)) || ((current_direction == Down) && (floor != lastPassedFloor-1)) {
					fmt.Println("\033[31m" + "Elevator: missed floor signal, entering error state" + "\033[0m")
					driver.SetMotorDirection(driver.MotorStop)
					elevatorError <- true
					state = errorState
					break
				}
				lastPassedFloor = floor
				driver.SetFloorIndicator(floor)
				state = atFloor
				deadlineTimer.Stop()
			case <-deadlineTimer.C:
				fmt.Println("\033[31m" + "Elevator: timeout while moving" + "\033[0m")
				driver.SetMotorDirection(driver.MotorStop)
				elevatorError <- true
				state = errorState
			}
		case reInitState:
			select {
			case floor := <-floorReached:
				driver.SetMotorDirection(driver.MotorStop)
				lastPassedFloor = floor
				driver.SetFloorIndicator(floor)
				deadlineTimer.Stop()
				state = atFloor
			case <-deadlineTimer.C:
				fmt.Println("\033[31m" + "Elevator: timeout while reinitializing" + "\033[0m")
				driver.SetMotorDirection(driver.MotorStop)
				elevatorError <- true
				state = errorState
			}
		case errorState:
			deadlineTimer.Stop()
			<-resumeAfterError
			if driver.GetFloorSensorSignal() == driver.InvalidFloor {
				startedMoving <- true
				switch current_direction {
				case Up:
					driver.SetMotorDirection(driver.MotorUp)
				case Down:
					driver.SetMotorDirection(driver.MotorDown)
				}
				deadlineTimer.Reset(deadlinePeriod)
				state = reInitState
			} else {
				lastPassedFloor = driver.GetFloorSensorSignal()
				state = atFloor
			}
			fmt.Println("\033[31m" + "Elevator: resuming operation after error" + "\033[0m")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
