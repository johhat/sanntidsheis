package main

import (
	"./com"
	driver "./driver"
	"./elevator"
	"./manager"
	"./networking"
	"log"
	"os"
	"os/signal"
	"runtime"
	"time"
)

func main() {

	go func() {
		for {
			log.Println("\033[33m"+"Number of active goroutines:", runtime.NumGoroutine(), "\033[0m")
			<-time.Tick(20 * time.Second)
		}
	}()

	clickEvent_chan := make(chan driver.ClickEvent)
	sensorEvent_chan := make(chan int)
	stopButtonChan := make(chan bool)

	completed_floor_chan := make(chan int)
	elev_error_chan := make(chan bool)
	floor_reached_chan := make(chan int)
	new_direction_chan := make(chan elevator.Direction_t)
	door_closed_chan := make(chan bool)
	readDir_chan := make(chan elevator.ReadDirection)
	readOrder_chan := make(chan elevator.ReadOrder)
	start_moving_chan := make(chan bool)
	passing_floor_chan := make(chan bool)

	sendMsgChan := make(chan com.Message)
	recvMsgChan := make(chan com.Message)
	connected := make(chan string)
	disconnected := make(chan string)
	setNetworkStatus := make(chan bool)
	resumeAfterError := make(chan bool)
	externalError := make(chan bool)

	driver.Init(clickEvent_chan, sensorEvent_chan, stopButtonChan)

	go elevator.Run(
		completed_floor_chan,
		elev_error_chan,
		floor_reached_chan,
		new_direction_chan,
		door_closed_chan,
		readDir_chan,
		readOrder_chan,
		start_moving_chan,
		passing_floor_chan,
		resumeAfterError,
		externalError)

	go manager.Run(
		sendMsgChan,
		recvMsgChan,
		connected,
		disconnected,
		readDir_chan,
		readOrder_chan,
		completed_floor_chan,
		door_closed_chan,
		clickEvent_chan,
		sensorEvent_chan,
		floor_reached_chan,
		start_moving_chan,
		new_direction_chan,
		passing_floor_chan,
		elev_error_chan,
		setNetworkStatus,
		resumeAfterError,
		stopButtonChan,
		externalError)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		driver.SetMotorDirection(driver.MotorStop)
		log.Fatal("[FATAL]\tUser terminated program")
	}()

	time.Sleep(500 * time.Millisecond) //Todo: Make network module default to disconnect state
	networking.NetworkLoop(sendMsgChan, recvMsgChan, connected, disconnected, setNetworkStatus)

}
