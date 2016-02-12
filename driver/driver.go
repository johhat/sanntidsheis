package main

/*
	#cgo CFLAGS: -std=c11
	#cgo LDFLAGS: -lcomedi -lm
	#include "channels.h"
	#include "io.h"
	#include "io.c"
*/
import "C"

import (
	. "./channels"
	"fmt"
	"log"
	"os"
)

const (
	NumFloors       = 4
	InvalidFloor    = -1
	InitFailureCode = 0
	MotorSpeed      = 2800
)

func main() {
	fmt.Println("Hello there")
	initialize()
	fmt.Println("Init-ed")
	fmt.Println("Floor sensor:", getFloorSensorSignal())
}

func initialize() {
	status := C.io_init()

	if status == InitFailureCode {
		log.Fatal("Hardware initialization failed. Exiting prorgram.")
		os.Exit(1)
	}

	setStopLamp(false)
	setDoorOpenLamp(false)
	setFloorIndicator(0)

	setBit(LIGHT_DOWN2)

	for i := 0; i < NumFloors; i++ {
		setCommandBtn(i, false)
		setOrderBtn(i, true, false)
	}
}

// Getters

func getOrderBtnSignal() {
	//Noop
}

func getCommandBtnSignal() {
	//Noop
}

func getFloorSensorSignal() int {
	switch {
	case readBit(SENSOR_FLOOR1) != 0:
		return 0
	case readBit(SENSOR_FLOOR2) != 0:
		return 1
	case readBit(SENSOR_FLOOR3) != 0:
		return 2
	case readBit(SENSOR_FLOOR4) != 0:
		return 3
	default:
		return -1
	}
}

func getStopSignal() bool {
	return readBit(STOP) == 1
}

func getObstructionSignal() bool {
	return readBit(OBSTRUCTION) == 1
}

// Setters

func setMotorDirection(direction int) {

	//NB: Ikke testet!

	switch direction {
	case -1:
		setBit(MOTORDIR)
		writeAnalog(MOTOR, MotorSpeed)
	case 0:
		writeAnalog(MOTOR, 0)
	case 1:
		clearBit(MOTORDIR)
		writeAnalog(MOTOR, MotorSpeed)
	}
}

func setFloorIndicator(floor int) {
	if floor&0x02 != 0 {
		setBit(LIGHT_FLOOR_IND1)
	} else {
		clearBit(LIGHT_FLOOR_IND1)
	}

	if floor&0x01 != 0 {
		setBit(LIGHT_FLOOR_IND2)

	} else {
		clearBit(LIGHT_FLOOR_IND2)
	}
}

func setCommandBtn(floor int, setTo bool) {

	var operation func(int)

	if setTo {
		operation = setBit
	} else {
		operation = clearBit
	}

	switch floor {
	case 0:
		operation(LIGHT_COMMAND1)
	case 1:
		operation(LIGHT_COMMAND2)
	case 2:
		operation(LIGHT_COMMAND3)
	case 3:
		operation(LIGHT_COMMAND4)
	default:
		log.Println("Tried to set command btn at non-existent floor", floor)
	}
}

func setOrderBtn(floor int, direction bool, setTo bool) {

	var operation func(int)

	if setTo {
		operation = setBit
	} else {
		operation = clearBit
	}

	if direction {
		switch floor {
		case 0:
			operation(LIGHT_UP1)
		case 1:
			operation(LIGHT_UP2)
		case 2:
			operation(LIGHT_UP3)
		default:
			log.Println("Tried to set order btn at non-existent floor", floor, direction)
		}
	} else {
		switch floor {
		case 1:
			operation(LIGHT_DOWN2)
		case 2:
			operation(LIGHT_DOWN3)
		case 3:
			operation(LIGHT_DOWN4)
		default:
			log.Println("Tried to set order btn at non-existent floor", floor, direction)
		}
	}

}

func setStopLamp(setTo bool) {
	if setTo {
		setBit(LIGHT_STOP)
	} else {
		clearBit(LIGHT_STOP)
	}
}

func setDoorOpenLamp(setTo bool) {
	if setTo {
		setBit(LIGHT_DOOR_OPEN)
	} else {
		clearBit(LIGHT_DOOR_OPEN)
	}
}

//Low level ops - consider moving to separate package
func setBit(channel int) {
	C.io_set_bit(C.int(channel))
}

func clearBit(channel int) {
	C.io_clear_bit(C.int(channel))
}

func writeAnalog(channel, value int) {
	C.io_write_analog(C.int(channel), C.int(value))
}

func readBit(channel int) int {
	return int(C.io_read_bit(C.int(channel)))
}

func readAnalog(channel int) int {
	return int(C.io_read_analog(C.int(channel)))
}
