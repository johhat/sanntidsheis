package driver

import (
	"./io"
	"log"
	"os"
	"time"
)

const (
	NumFloors    = 4
	NumBtnTypes  = 3
	InvalidFloor = -1
	MotorSpeed   = 2800
	PollInterval = 1 * time.Millisecond
)

func pollFloorSensor(sensorEventChan chan<- int) {

	state := -1

	for {
		sensorSignal := GetFloorSensorSignal()

		if state != sensorSignal {
			state = sensorSignal
			sensorEventChan <- state
		}
		time.Sleep(PollInterval)
	}
}

func pollButtons(clickEventChan chan<- ClickEvent) {

	var isPressed [NumBtnTypes][NumFloors]bool

	for {
		for f := 0; f < NumFloors; f++ {

			for btn := 0; btn < NumBtnTypes; btn++ {
				if isPressed[BtnType(btn)][f] != getBtnSignal(f, BtnType(btn)) {
					isPressed[BtnType(btn)][f] = !isPressed[BtnType(btn)][f]

					if isPressed[BtnType(btn)][f] {
						clickEventChan <- ClickEvent{f, BtnType(btn)}
					}
				}
			}
		}
		time.Sleep(PollInterval)
	}
}

func pollStopButton(stopButtonChan chan<- bool) {
	isPressed := getStopBtnSignal()

	for {
		if isPressed != getStopBtnSignal() {
			isPressed = !isPressed

			if isPressed {
				stopButtonChan <- true
			}
		}
		time.Sleep(PollInterval)
	}
}

func init() {

	err := io.Init()

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	SetStopLamp(false)
	SetDoorOpenLamp(false)
	clearBtnLamps()

	SetMotorDirection(MotorDown)

	timeout := time.After(10 * time.Second)

	for GetFloorSensorSignal() == InvalidFloor {
		select {
		case <-timeout:
			log.Fatal("Timeout in driver. Did not get to valid floor in time.")
			os.Exit(1)
		default:
		}
	}

	SetFloorIndicator(GetFloorSensorSignal())

	SetMotorDirection(MotorStop)
}

func Init(clickEventChan chan<- ClickEvent, sensorEventChan chan<- int, stopButtonChan chan<- bool) {
	go pollFloorSensor(sensorEventChan)
	go pollButtons(clickEventChan)
	go pollStopButton(stopButtonChan)
}

func clearBtnLamps() {
	for f := 0; f < NumFloors; f++ {
		for btn := 0; btn < NumBtnTypes; btn++ {
			SetBtnLamp(f, BtnType(btn), false)
		}
	}
}

func getBtnSignal(floor int, button BtnType) bool {

	if floor < 0 || floor >= NumFloors {
		log.Println("Tried to get signal form non-existent floor")
		return false
	}

	var buttonChannels = [NumFloors][NumBtnTypes]int{
		[NumBtnTypes]int{BUTTON_UP1, BUTTON_DOWN1, BUTTON_COMMAND1},
		[NumBtnTypes]int{BUTTON_UP2, BUTTON_DOWN2, BUTTON_COMMAND2},
		[NumBtnTypes]int{BUTTON_UP3, BUTTON_DOWN3, BUTTON_COMMAND3},
		[NumBtnTypes]int{BUTTON_UP4, BUTTON_DOWN4, BUTTON_COMMAND4},
	}

	switch button {
	case Up, Down, Command:
		return io.ReadBit(buttonChannels[floor][int(button)]) != 0
	default:
		log.Println("Tried to get signal form non-existent btn")
		return false
	}
}

func GetFloorSensorSignal() int {
	switch {
	case io.ReadBit(SENSOR_FLOOR1) != 0:
		return 0
	case io.ReadBit(SENSOR_FLOOR2) != 0:
		return 1
	case io.ReadBit(SENSOR_FLOOR3) != 0:
		return 2
	case io.ReadBit(SENSOR_FLOOR4) != 0:
		return 3
	default:
		return -1
	}
}

func getStopBtnSignal() bool {
	return io.ReadBit(STOP) == 1
}

func SetMotorDirection(direction MotorDirection) {
	switch direction {
	case MotorDown:
		io.SetBit(MOTORDIR)
		io.WriteAnalog(MOTOR, MotorSpeed)
	case MotorStop:
		io.WriteAnalog(MOTOR, 0)
	case MotorUp:
		io.ClearBit(MOTORDIR)
		io.WriteAnalog(MOTOR, MotorSpeed)
	}
}

func SetFloorIndicator(floor int) {
	if floor&0x02 != 0 {
		io.SetBit(LIGHT_FLOOR_IND1)
	} else {
		io.ClearBit(LIGHT_FLOOR_IND1)
	}

	if floor&0x01 != 0 {
		io.SetBit(LIGHT_FLOOR_IND2)

	} else {
		io.ClearBit(LIGHT_FLOOR_IND2)
	}
}

func SetBtnLamp(floor int, btn BtnType, setTo bool) {

	var lightChannels = [NumFloors][NumBtnTypes]int{
		[NumBtnTypes]int{LIGHT_UP1, LIGHT_DOWN1, LIGHT_COMMAND1},
		[NumBtnTypes]int{LIGHT_UP2, LIGHT_DOWN2, LIGHT_COMMAND2},
		[NumBtnTypes]int{LIGHT_UP3, LIGHT_DOWN3, LIGHT_COMMAND3},
		[NumBtnTypes]int{LIGHT_UP4, LIGHT_DOWN4, LIGHT_COMMAND4},
	}

	switch btn {
	case Up, Down, Command:
		if setTo {
			io.SetBit(lightChannels[floor][int(btn)])
		} else {
			io.ClearBit(lightChannels[floor][int(btn)])
		}
	default:
		log.Println("Btn type failure in SetBtnLamp. Floor: ", floor)
	}
}

func SetStopLamp(setTo bool) {
	if setTo {
		io.SetBit(LIGHT_STOP)
	} else {
		io.ClearBit(LIGHT_STOP)
	}
}

func SetDoorOpenLamp(setTo bool) {
	if setTo {
		io.SetBit(LIGHT_DOOR_OPEN)
	} else {
		io.ClearBit(LIGHT_DOOR_OPEN)
	}
}
