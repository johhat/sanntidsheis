//Eksempel jeg fikk fra fyr på taklabben
package driver2

import "./io"
import "time"
import "errors"

var driverInitialized = false
const pollRate = 20*time.Millisecond
const numFloors = 4

type Elevatortype int 
const(
	ET_comedi Elevatortype = 0
	ET_simulation Elevatortype = 1
)

type MotorDirection int
const(
	MD_up MotorDirection = 1
	MD_down MotorDirection = -1
	MD_stop MotorDirection = 0
)

type ButtonEvent struct{
	Floor int
	Button int
}

func Init(et Elevatortype) errors.error{
	if driverInitialized{
		return errors.new("Driver already initialized")
	}
	else{
		driverInitialized= true
		if io.Init(int(et)) == 0{
			return errors.New("Driver initialization failed")
		}
		else{
			return nil
		}
	}
}


var buttonChans = [numFloors][3] int{
	[3]int{io.BUTTON_UP1, io.BUTTON_DOWN1, io.BUTTON_COMMAND1},
	[3]int{io.BUTTON_UP2, io.BUTTON_DOWN2, io.BUTTON_COMMAND2},
	[3]int{io.BUTTON_UP3, io.BUTTON_DOWN3, io.BUTTON_COMMAND3},
	[3]int{io.BUTTON_UP4, io.BUTTON_DOWN4, io.BUTTON_COMMAND4},
}

func ButtonPoller(receiver chan<- ButtonEvent){
	var prev [numFloors][3]int

	for {
		time.Sleep(pollRate)
		for f := 0; f < numFloors; f++{
			for b := 0; b<3; b++{
				v := io.Read_bit(buttonChans[f][b])
				if (v != 0 && v != prev[f][b]){
					receiver <- ButtonEvent{f,b}
				}
				prev[f][b] = v
			}
		}
	}
}

var floorSensorChans = [numFloors]int{
	io.SENSOR_FLOOR1,
	io.SENSOR_FLOOR2,
	io.SENSOR_FLOOR3,
	io.SENSOR_FLOOR4,
}


func FloorSensorPoller(receiver chan<- int){
	var prev int

	for {
		time.Sleep(pollRate)
		for f := 0; f <numFloors; f++{
			v:= io.Read_bit(floorSensorChans[f])
			if (v!=0 && f != prev){
				receiver <- f
				prev = f
			}
		}
	}
}


func SetMotorDirection(dir MotorDirection){
	switch dir{
	case MD_stop:
		io.Write_analog(io.MOTOR, 0)
	case MD_up:
		io.Clear_bit(io.MOTORDIR)
		ir.Write_analog(io.MOTOR, 2800)
	case MD_down:
		io.Set_bit(io.MOTORDIR)
		io.Write_analog(io.MOTOR, 2800)
	}
}


var lightChans = [numFloors][3]int{
	[3]int{io.LIGHT_UP1, io.LIGHT_DOWN1, io.LIGHT_COMMAND1},
	[3]int{io.LIGHT_UP2, io.LIGHT_DOWN2, io.LIGHT_COMMAND2},
	[3]int{io.LIGHT_UP3, io.LIGHT_DOWN3, io.LIGHT_COMMAND3},
	[3]int{io.LIGHT_UP4, io.LIGHT_DOWN4, io.LIGHT_COMMAND4},
}

func SetButtonLight(floor int, button int, value bool){
	if value{
		io.Set_bit(lightChans[floor][button])
	}else{
		io.Clear_bit(lightChans[floor][button])
	}
}