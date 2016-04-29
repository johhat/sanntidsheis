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
	"time"
)

func main() {

	clickEvent_chan := make(chan driver.ClickEvent)
	sensorEvent_chan := make(chan int)

	completed_floor_chan := make(chan int)
	elev_error_chan := make(chan bool)
	floor_reached_chan := make(chan int)
	new_order_chan := make(chan driver.ClickEvent)
	new_direction_chan := make(chan elevator.Direction_t)
	door_closed_chan := make(chan bool)
	readDir_chan := make(chan elevator.ReadDirection)
	readOrder_chan := make(chan elevator.ReadOrder)
	networking_timeout := make(chan bool)
	start_moving_chan := make(chan bool)
	passing_floor_chan := make(chan bool)

	sendMsgChan := make(chan com.Message)
	recvMsgChan := make(chan com.Message)
	connected := make(chan string)
	disconnected := make(chan string)
	disconnectFromNetwork := make(chan bool)
	reconnectToNetwork := make(chan bool)

	driver.Init(clickEvent_chan, sensorEvent_chan)

	go elevator.Run(
		completed_floor_chan,
		elev_error_chan,
		floor_reached_chan,
		new_order_chan,
		new_direction_chan,
		door_closed_chan,
		readDir_chan,
		readOrder_chan,
		start_moving_chan,
		passing_floor_chan)

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
		passing_floor_chan,
		elev_error_chan,
		disconnectFromNetwork,
		reconnectToNetwork,
		networking_timeout)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		driver.SetMotorDirection(driver.MotorStop)
		log.Fatal("[FATAL]\tUser terminated program")
	}()
	time.Sleep(500 * time.Millisecond) //Todo: Make network module default to disconnect state
	networking.NetworkLoop(sendMsgChan, recvMsgChan, connected, disconnected, disconnectFromNetwork, reconnectToNetwork)

}
