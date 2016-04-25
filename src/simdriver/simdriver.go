package simdriver

// Simulator mk. II

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

//TODO: close connection correctly

const remoteIp = "localhost:15657"

var connection net.Conn
var state int

const (
	NumFloors    = 4
	NumBtnTypes  = 3
	InvalidFloor = -1
	MotorSpeed   = 2800
	PollInterval = 2 * time.Millisecond
)

type BtnType int
type MotorDirection int

type ClickEvent struct {
	Floor int
	Type  BtnType
}

func (event ClickEvent) String() string {
	return fmt.Sprintf("%s button click at floor %d", event.Type, event.Floor)
}

const (
	Up BtnType = iota
	Down
	Command
)

func (btn BtnType) String() string {
	switch btn {
	case Up:
		return "Up"
	case Down:
		return "Down"
	case Command:
		return "Command"
	default:
		return fmt.Sprintf("Btn(%d)", btn)
	}
}

const (
	MotorUp MotorDirection = iota
	MotorStop
	MotorDown
)

func GetCurrentFloor() int {
	return state
}

func (direction MotorDirection) String() string {
	switch direction {
	case MotorUp:
		return "Motor up"
	case MotorStop:
		return "Motor stop"
	case MotorDown:
		return "Motor down"
	default:
		return fmt.Sprintf("MotorDirection(%d)", direction)
	}
}

func poll(sensorEventChan chan int, clickEventChan chan ClickEvent) {

	state = GetFloorSensorSignal()
	var isPressed [NumBtnTypes][NumFloors]bool

	for {
		//Poll floor sensors
		sensorSignal := GetFloorSensorSignal()

		if state != sensorSignal {
			state = sensorSignal
			sensorEventChan <- state
		}

		//Poll buttons
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

func BasicElevator() {

	SetMotorDirection(MotorUp)

	for {
		switch {
		case GetFloorSensorSignal() == 0:
			SetMotorDirection(MotorUp)
		case GetFloorSensorSignal() == NumFloors-1:
			SetMotorDirection(MotorDown)
		case getObstructionSignal():
			SetMotorDirection(MotorStop)
			os.Exit(1)
		case getStopSignal():
			SetMotorDirection(MotorStop)
			os.Exit(1)
		}
	}
}

func Xinit() {

	boolPtr := flag.Bool("dryRun", false, "Use flag dryRun to execute program without init of elevator sim")

	flag.Parse()

	if !*boolPtr {
		conn, err := net.Dial("tcp", remoteIp)
		for {
			if err != nil {
				log.Printf("TCP dial to %s failed", remoteIp)
				time.Sleep(500 * time.Millisecond)
				conn, err = net.Dial("tcp", remoteIp)
			} else {
				break
			}
		}
		connection = conn

		SetStopLamp(false)
		SetDoorOpenLamp(false)
		clearBtnLamps()
		SetMotorDirection(MotorDown)

		for GetFloorSensorSignal() == InvalidFloor {
			time.Sleep(10 * time.Millisecond)
			//TODO: Add timeout
		}
		SetFloorIndicator(GetFloorSensorSignal())
		SetMotorDirection(MotorStop)
	} else {
		log.Println("simdriver not initialized as flag --dryRun is present")
	}
}

func Init(clickEventChan chan ClickEvent, sensorEventChan chan int) {
	go poll(sensorEventChan, clickEventChan)
	time.Sleep(10 * time.Millisecond)

}

func clearBtnLamps() {
	for f := 0; f < NumFloors; f++ {
		for btn := 0; btn < NumBtnTypes; btn++ {
			SetBtnLamp(f, BtnType(btn), false)
		}
	}
}

// Getters
func getBtnSignal(floor int, button BtnType) bool {

	if floor < 0 || floor >= NumFloors {
		log.Println("Tried to get signal from non-existent floor")
		return false
	}

	/*var buttonChannels = [NumFloors][NumBtnTypes]int{
		[NumBtnTypes]int{BUTTON_UP1, BUTTON_DOWN1, BUTTON_COMMAND1},
		[NumBtnTypes]int{BUTTON_UP2, BUTTON_DOWN2, BUTTON_COMMAND2},
		[NumBtnTypes]int{BUTTON_UP3, BUTTON_DOWN3, BUTTON_COMMAND3},
		[NumBtnTypes]int{BUTTON_UP4, BUTTON_DOWN4, BUTTON_COMMAND4},
	}*/

	switch button {
	case Up, Down, Command:
		//return ReadBit(buttonChannels[floor][int(button)]) != 0
		_, err := connection.Write([]byte{6, byte(int(button)), byte(floor), 0})
		if err != nil {
			log.Fatal("write error:", err)
		}
		buf := make([]byte, 4)
		_, err = io.ReadFull(connection, buf)
		if buf[1] == 1 {
			return true
		} else {
			return false
		}
	default:
		log.Println("Tried to get signal from non-existent btn")
		return false
	}
}

func GetFloorSensorSignal() int {
	/*switch {
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
	}*/

	_, err := connection.Write([]byte{7, 0, 0, 0})
	if err != nil {
		log.Fatal("write error:", err)
	}

	buf := make([]byte, 4)

	_, err = io.ReadFull(connection, buf)
	if err != nil {
		log.Fatal("read error:", err)
	}
	if buf[0] != 7 {
		log.Println("Returned floor sensor message is not valid")
	}
	if buf[1] == 1 {
		return int(buf[2])
	} else {
		return -1
	}
}

func getStopSignal() bool {
	//return ReadBit(STOP) == 1
	_, err := connection.Write([]byte{8, 0, 0, 0})
	if err != nil {
		log.Fatal("write error:", err)
	}
	buf := make([]byte, 4)
	_, err = io.ReadFull(connection, buf)
	if err != nil {
		log.Fatal("read error:", err)
	}
	if buf[1] == 1 {
		return true
	} else {
		return false
	}
}

func getObstructionSignal() bool {
	//return ReadBit(OBSTRUCTION) == 1
	_, err := connection.Write([]byte{9, 0, 0, 0})
	if err != nil {
		log.Fatal("write error:", err)
	}
	buf := make([]byte, 4)
	_, err = io.ReadFull(connection, buf)
	if err != nil {
		log.Fatal("read error:", err)
	}
	if buf[1] == 1 {
		return true
	} else {
		return false
	}
}

// Setters

func SetMotorDirection(direction MotorDirection) {
	switch direction {
	case MotorDown:
		//SetBit(MOTORDIR)
		//WriteAnalog(MOTOR, MotorSpeed)
		_, err := connection.Write([]byte{1, 255, 0, 0})
		if err != nil {
			log.Fatal("write error:", err)
		}
	case MotorStop:
		//WriteAnalog(MOTOR, 0)
		_, err := connection.Write([]byte{1, 0, 0, 0})
		if err != nil {
			log.Fatal("write error:", err)
		}
	case MotorUp:
		//ClearBit(MOTORDIR)
		//WriteAnalog(MOTOR, MotorSpeed)
		_, err := connection.Write([]byte{1, 1, 0, 0})
		if err != nil {
			log.Fatal("write error:", err)
		}
	}
}

func SetFloorIndicator(floor int) {
	/*if floor&0x02 != 0 {
		SetBit(LIGHT_FLOOR_IND1)
	} else {
		ClearBit(LIGHT_FLOOR_IND1)
	}

	if floor&0x01 != 0 {
		SetBit(LIGHT_FLOOR_IND2)

	} else {
		ClearBit(LIGHT_FLOOR_IND2)
	}*/

	_, err := connection.Write([]byte{3, byte(floor), 0, 0})
	if err != nil {
		log.Fatal("write error:", err)
	}
}

func SetBtnLamp(floor int, btn BtnType, setTo bool) {

	/*var lightChannels = [NumFloors][NumBtnTypes]int{
		[NumBtnTypes]int{LIGHT_UP1, LIGHT_DOWN1, LIGHT_COMMAND1},
		[NumBtnTypes]int{LIGHT_UP2, LIGHT_DOWN2, LIGHT_COMMAND2},
		[NumBtnTypes]int{LIGHT_UP3, LIGHT_DOWN3, LIGHT_COMMAND3},
		[NumBtnTypes]int{LIGHT_UP4, LIGHT_DOWN4, LIGHT_COMMAND4},
	}*/
	switch btn {
	case Up, Down, Command:
		if setTo {
			//SetBit(lightChannels[floor][int(btn)])
			_, err := connection.Write([]byte{2, byte(int(btn)), byte(floor), 1})
			if err != nil {
				log.Fatal("write error:", err)
			}
		} else {
			//ClearBit(lightChannels[floor][int(btn)])
			_, err := connection.Write([]byte{2, byte(int(btn)), byte(floor), 0})
			if err != nil {
				log.Fatal("write error:", err)
			}
		}

	default:
		log.Println("Btn type failure in setBtnLamp. Floor: ", floor)
	}
}

func SetStopLamp(setTo bool) {
	if setTo {
		//SetBit(LIGHT_STOP)
		_, err := connection.Write([]byte{5, 1, 0, 0})
		if err != nil {
			log.Fatal("write error:", err)
		}
	} else {
		//ClearBit(LIGHT_STOP)
		_, err := connection.Write([]byte{5, 0, 0, 0})
		if err != nil {
			log.Fatal("write error:", err)
		}
	}
}

func SetDoorOpenLamp(setTo bool) {
	if setTo {
		//SetBit(LIGHT_DOOR_OPEN)
		_, err := connection.Write([]byte{4, 1, 0, 0})
		if err != nil {
			log.Fatal("write error:", err)
		}

	} else {
		//ClearBit(LIGHT_DOOR_OPEN)
		_, err := connection.Write([]byte{4, 0, 0, 0})
		if err != nil {
			log.Fatal("write error:", err)
		}
	}
}
