package driver

import (
	. "./channels"
	. "./elevatorIo"
	"log"
	"os"
	"time"
)

type BtnType int

type ClickEvent struct {
	Floor int
	Type  BtnType
}

const (
	NumFloors    = 4
	InvalidFloor = -1
	MotorSpeed   = 2800
	PollInterval = 20 * time.Millisecond
)

const (
	Up BtnType = iota
	Down
	Command
)

func pollFloorSensor(sensorEventChan chan int) {

	state := -1

	for {
		sensorSignal := getFloorSensorSignal()

		if state != sensorSignal {
			if state > -1 {
				log.Println("Exited floor", state)
			} else {
				log.Println("Entered floor", sensorSignal)
			}

			state = sensorSignal

			sensorEventChan <- state
		}
		time.Sleep(PollInterval)
	}
}

func pollCommandButtons(clickEventChan chan ClickEvent) {

	var isPressed [NumFloors]bool

	for {
		for i := 0; i < NumFloors; i++ {
			if isPressed[i] != getCommandBtnSignal(i) {
				isPressed[i] = !isPressed[i]
				if isPressed[i] {
					clickEventChan <- ClickEvent{i, Command}
				}
			}
		}
		time.Sleep(PollInterval)
	}
}

func pollOrderButtons(clickEventChan chan ClickEvent) {

	var isPressed [2][NumFloors]bool

	for {
		for i := 0; i < NumFloors; i++ {

			if isPressed[Up][i] != getOrderBtnSignal(i, true) {

				isPressed[Up][i] = !isPressed[Up][i]

				if isPressed[Up][i] {
					clickEventChan <- ClickEvent{i, Up}
				}
			}

			if isPressed[Down][i] != getOrderBtnSignal(i, false) {

				isPressed[Down][i] = !isPressed[Down][i]

				if isPressed[Down][i] {
					clickEventChan <- ClickEvent{i, Down}
				}
			}
		}
		time.Sleep(PollInterval)
	}
}

func basicElevator() {

	setMotorDirection(1)

	for {
		switch {
		case getFloorSensorSignal() == 0:
			setMotorDirection(1)
		case getFloorSensorSignal() == 3:
			setMotorDirection(-1)
		case getObstructionSignal():
			setMotorDirection(0)
			os.Exit(1)
		case getStopSignal():
			setMotorDirection(0)
			os.Exit(1)
		}
	}
}

func Initialize(clickEventChan chan ClickEvent, sensorEventChan chan int) {
	err := InitializeElevatorIo()

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	setStopLamp(false)
	setDoorOpenLamp(false)
	setFloorIndicator(0)

	for i := 0; i < NumFloors; i++ {
		setCommandLamp(i, false)
	}

	for i := 0; i < NumFloors-1; i++ {
		setOrderLamp(i, true, false)
		setOrderLamp(i+1, false, false)
	}

	go pollFloorSensor(sensorEventChan)
	go pollCommandButtons(clickEventChan)
	go pollOrderButtons(clickEventChan)

	basicElevator()
}

// Getters

func getOrderBtnSignal(floor int, direction bool) bool {
	if direction {
		switch floor {
		case 0:
			return ReadBit(BUTTON_UP1) != 0
		case 1:
			return ReadBit(BUTTON_UP2) != 0
		case 2:
			return ReadBit(BUTTON_UP3) != 0
		case 3:
			return false
		default:
			log.Println("Tried to get signal from non-existent floor")
			return false
		}
	} else {
		switch floor {
		case 0:
			return false
		case 1:
			return ReadBit(BUTTON_DOWN2) != 0
		case 2:
			return ReadBit(BUTTON_DOWN3) != 0
		case 3:
			return ReadBit(BUTTON_DOWN4) != 0
		default:
			log.Println("Tried to get signal from non-existent floor")
			return false
		}
	}
}

func getCommandBtnSignal(floor int) bool {
	switch floor {
	case 0:
		return ReadBit(BUTTON_COMMAND1) != 0
	case 1:
		return ReadBit(BUTTON_COMMAND2) != 0
	case 2:
		return ReadBit(BUTTON_COMMAND3) != 0
	case 3:
		return ReadBit(BUTTON_COMMAND4) != 0
	default:
		log.Println("Tried to get command btn signal at non-existent floor:", floor)
		return false
	}
}

func getFloorSensorSignal() int {
	switch {
	case ReadBit(SENSOR_FLOOR1) != 0:
		return 0
	case ReadBit(SENSOR_FLOOR2) != 0:
		return 1
	case ReadBit(SENSOR_FLOOR3) != 0:
		return 2
	case ReadBit(SENSOR_FLOOR4) != 0:
		return 3
	default:
		return -1
	}
}

func getStopSignal() bool {
	return ReadBit(STOP) == 1
}

func getObstructionSignal() bool {
	return ReadBit(OBSTRUCTION) == 1
}

// Setters

func setMotorDirection(direction int) {
	switch {
	case direction < 0:
		SetBit(MOTORDIR)
		WriteAnalog(MOTOR, MotorSpeed)
	case direction == 0:
		WriteAnalog(MOTOR, 0)
	case direction > 0:
		ClearBit(MOTORDIR)
		WriteAnalog(MOTOR, MotorSpeed)
	}
}

func setFloorIndicator(floor int) {
	if floor&0x02 != 0 {
		SetBit(LIGHT_FLOOR_IND1)
	} else {
		ClearBit(LIGHT_FLOOR_IND1)
	}

	if floor&0x01 != 0 {
		SetBit(LIGHT_FLOOR_IND2)

	} else {
		ClearBit(LIGHT_FLOOR_IND2)
	}
}

func setCommandLamp(floor int, setTo bool) {

	var operation func(int)

	if setTo {
		operation = SetBit
	} else {
		operation = ClearBit
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

func setOrderLamp(floor int, direction bool, setTo bool) {

	var operation func(int)

	if setTo {
		operation = SetBit
	} else {
		operation = ClearBit
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
		SetBit(LIGHT_STOP)
	} else {
		ClearBit(LIGHT_STOP)
	}
}

func setDoorOpenLamp(setTo bool) {
	if setTo {
		SetBit(LIGHT_DOOR_OPEN)
	} else {
		ClearBit(LIGHT_DOOR_OPEN)
	}
}
