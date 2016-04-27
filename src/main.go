package main

import (
	"./networking"
	"./manager"
	"./elevator"
	"./simdriver"
	"os"
	"os/signal"
	"log"
	"./com"
)

func main() {

	var clickEvent_chan = make(chan simdriver.ClickEvent)
	var sensorEvent_chan = make(chan int)

	var completed_floor_chan = make(chan int)
	var elev_error_chan = make(chan bool)
	var floor_reached_chan = make(chan int)
	var new_order_chan = make(chan simdriver.ClickEvent)
	var new_direction_chan = make(chan elevator.Direction_t)
	var door_closed_chan = make(chan bool)
	var readDir_chan = make(chan elevator.ReadDirection)
	var readOrder_chan = make(chan elevator.ReadOrder)
	var deletes_chan = make(chan elevator.DeleteOp)
	var drop_conn_chan = make(chan bool)
	var networking_timeout = make(chan bool)
	var start_moving_chan = make(chan bool)
	var passing_floor_chan = make(chan bool)
	

	var sendMsgChan = make(chan com.Message)
	var recvMsgChan = make(chan com.Message)
	var connected = make(chan string)
	var disconnected = make(chan string)

	simdriver.Init(clickEvent_chan, sensorEvent_chan)

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
		passing_floor_chan,
		deletes_chan)

	go networking.NetworkLoop(sendMsgChan, recvMsgChan, connected, disconnected)

	c := make(chan os.Signal)
    signal.Notify(c, os.Interrupt)
    go func() {
        <- c
        simdriver.SetMotorDirection(simdriver.MotorStop)
        log.Fatal("[FATAL]\tUser terminated program")
    }()

    manager.Run(
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
		drop_conn_chan,
		networking_timeout)

}
