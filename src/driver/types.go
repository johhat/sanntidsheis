package driver

import (
	"fmt"
)

type BtnType int

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

type MotorDirection int

const (
	MotorUp MotorDirection = iota
	MotorStop
	MotorDown
)

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

type ClickEvent struct {
	Floor int
	Type  BtnType
}

func (event ClickEvent) String() string {
	return fmt.Sprintf("%s button click at floor %d", event.Type, event.Floor)
}
