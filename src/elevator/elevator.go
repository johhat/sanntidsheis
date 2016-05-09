package elevator

import (
	"../driver"
	"fmt"
	"log"
	"time"
)

const (
	deadlinePeriod = 3 * time.Second
	doorPeriod     = 3 * time.Second
)

var currentDirection Direction = Up

func GetCurrentDirection() Direction {
	return currentDirection
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
			fmt.Println("Elevator: received external error")
			driver.SetMotorDirection(driver.MotorStop)
			elevatorError <- true
			state = errorState
		default:
		}
		switch state {
		case atFloor:
			readOrders <- ReadOrder{driver.ClickEvent{lastPassedFloor, driver.Command}, readResult}
			internal0rderAtThisFloor := <-readResult
			readOrders <- ReadOrder{driver.ClickEvent{lastPassedFloor, currentDirection.toBtnType()}, readResult}
			orderForwardAtThisFloor := <-readResult

			if internal0rderAtThisFloor || orderForwardAtThisFloor {
				isPassingFloor = false
				driver.SetMotorDirection(driver.MotorStop)
				driver.SetDoorOpenLamp(true)
				completedFloor <- lastPassedFloor
				deadlineTimer.Stop()
				doorTimer.Reset(doorPeriod)
				state = doorOpen
				break
			}

			readDirection <- ReadDirection{lastPassedFloor, currentDirection, IsOrderAhead, readResult}
			orderAhead := <-readResult

			if orderAhead {
				startedMoving <- true
				if isPassingFloor {
					passingFloor <- true
				}
				switch currentDirection {
				case Up:
					driver.SetMotorDirection(driver.MotorUp)
				case Down:
					driver.SetMotorDirection(driver.MotorDown)
				}
				deadlineTimer.Reset(deadlinePeriod)
				state = movingBetween
				isPassingFloor = true
				break

			}

			readDirection <- ReadDirection{lastPassedFloor, currentDirection, IsOrderBehind, readResult}
			orderBehind := <-readResult
			readOrders <- ReadOrder{driver.ClickEvent{lastPassedFloor, currentDirection.OppositeDirection().toBtnType()}, readResult}
			orderBackwardAtThisFloor := <-readResult

			if orderBehind || orderBackwardAtThisFloor {
				currentDirection = currentDirection.OppositeDirection()
				newDirection <- currentDirection
			}

		case doorOpen:
			<-doorTimer.C
			driver.SetDoorOpenLamp(false)
			state = atFloor
			doorClosed <- true
		case movingBetween:
			select {
			case floor := <-floorReached:
				if ((currentDirection == Up) && (floor != lastPassedFloor+1)) || ((currentDirection == Down) && (floor != lastPassedFloor-1)) {
					fmt.Println("Elevator: missed floor signal, entering error state")
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
				fmt.Println("Elevator: timeout while moving")
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
				fmt.Println("Elevator: timeout while reinitializing")
				driver.SetMotorDirection(driver.MotorStop)
				elevatorError <- true
				state = errorState
			}
		case errorState:
			deadlineTimer.Stop()
			<-resumeAfterError
			if driver.GetFloorSensorSignal() == driver.InvalidFloor {
				startedMoving <- true
				switch currentDirection {
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
		}
		time.Sleep(5 * time.Millisecond)
	}
}
