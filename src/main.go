package main

import (
	"./com"
	"./driver"
	"./elevator"
	"./manager"
	"./networking"
	"log"
	"os"
	"os/signal"
)

func main() {

	// Driver channels
	clickEvent := make(chan driver.ClickEvent)
	sensorEvent := make(chan int)
	stopBtnEvent := make(chan bool)

	// Elevator event channels
	completedFloor := make(chan int)
	floorReached := make(chan int)
	newDirection := make(chan elevator.Direction)
	doorClosed := make(chan bool)
	startedMoving := make(chan bool)
	passingFloor := make(chan bool)

	// Elevator error channels
	elevatorError := make(chan bool)
	resumeAfterError := make(chan bool)
	externalError := make(chan bool)

	// Elevator order channels
	readDirection := make(chan elevator.ReadDirection)
	readOrder := make(chan elevator.ReadOrder)

	// Network channels
	sendMsg := make(chan com.Message)
	recvMsg := make(chan com.Message)
	connected := make(chan string)
	disconnected := make(chan string)
	setNetworkStatus := make(chan bool)

	driver.Init(clickEvent, sensorEvent, stopBtnEvent)

	go elevator.Run(
		completedFloor,
		floorReached,
		newDirection,
		doorClosed,
		startedMoving,
		passingFloor,
		elevatorError,
		resumeAfterError,
		externalError,
		readDirection,
		readOrder)

	go manager.Run(
		clickEvent,
		sensorEvent,
		stopBtnEvent,
		completedFloor,
		floorReached,
		newDirection,
		doorClosed,
		startedMoving,
		passingFloor,
		elevatorError,
		resumeAfterError,
		externalError,
		readDirection,
		readOrder,
		sendMsg,
		recvMsg,
		connected,
		disconnected,
		setNetworkStatus)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		driver.SetMotorDirection(driver.MotorStop)
		log.Fatal("[FATAL]\tUser terminated program")
	}()

	networking.Run(sendMsg, recvMsg, connected, disconnected, setNetworkStatus)
}
