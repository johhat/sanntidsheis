package elevator

import (
    "time"
    "../simdriver"
)

const deadline_period = 5 * time.Second //Juster denne
const door_period = 3 * time.Second
var last_passed_floor int

type state_t int
const (
    atFloor state_t = iota
    doorOpen
    moving
)

type direction_t int
const (
	up direction_t = iota
	down
)

// Tilstand som viser nåværende retning?

func Run(
	completed_floor  chan <- int,
	missed_deadline  chan <- bool,
	floor_reached    <- chan int,
    new_target_floor <- chan int){

	deadline_timer := time.NewTimer(deadline_period)
    deadline_timer.Stop()

    door_timer := time.NewTimer(door_period)
    door_timer.Stop()

    state := atFloor

	for{
		select{
			case <- door_timer.C:
			case <- deadline_timer.C:

		}
	}
}

func orderAhead(currentFloor int, direction){

}

