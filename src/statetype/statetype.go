package statetype

import (
	"../elevator"
	"../simdriver"
	"fmt"
	"os"
	"strconv"
)

type State struct {
	LastPassedFloor int
	Direction       elevator.Direction_t
	Moving          bool
	Orders          Orderset
	Valid           bool
	SequenceNumber  int
	DoorOpen        bool
}

const (
	stopTime         float32 = 3
	floorTravelTime  float32 = 2 //2.2+floorPassingTime
	floorPassingTime float32 = 0.385
	movingPenalty    float32 = floorTravelTime / 2
	doorOpenPenalty  float32 = stopTime / 2
)

func (orders Orderset) RestoreInternalOrders() {
	for floor := 0; floor < simdriver.NumFloors; floor++ {
		if _, err := os.Stat("internalOrder" + strconv.Itoa(floor)); !os.IsNotExist(err) {
			orders[simdriver.Command][floor] = true
		}
	}
}

func SaveInternalOrder(floor int) {
	_, err := os.Create("internalOrder" + strconv.Itoa(floor))
	if err != nil {
		fmt.Println(err)
	}
}

func (orders Orderset) IsOrder(event simdriver.ClickEvent) bool {
	if event.Floor < 0 || event.Floor > simdriver.NumFloors-1 {
		//fmt.Println("Attempted to check order for non-existing floor")
		return false
	} else if event.Type == simdriver.Up && event.Floor == simdriver.NumFloors-1 {
		return false
	} else if event.Type == simdriver.Down && event.Floor == 0 {
		return false
	}
	return orders[event.Type][event.Floor]
}

func (orders Orderset) IsOrderAhead(currentFloor int, direction elevator.Direction_t) bool {
	for _, buttonOrders := range orders {
		for floor, isSet := range buttonOrders {
			if (direction == elevator.Up) && (floor > currentFloor) && isSet {
				return true
			} else if (direction == elevator.Down) && (floor < currentFloor) && isSet {
				return true
			}
		}
	}
	return false
}

func (orders Orderset) IsOrderBehind(currentFloor int, direction elevator.Direction_t) bool {
	return orders.IsOrderAhead(currentFloor, direction.OppositeDirection())
}

func (orders Orderset) Init() {
	for floor := 0; floor < simdriver.NumFloors; floor++ {
		if floor == 0 {
			orders[simdriver.Up][floor] = false
			orders[simdriver.Command][floor] = false
		} else if floor == simdriver.NumFloors-1 {
			orders[simdriver.Down][floor] = false
			orders[simdriver.Command][floor] = false
		} else {
			orders[simdriver.Up][floor] = false
			orders[simdriver.Down][floor] = false
			orders[simdriver.Command][floor] = false
		}
	}
}

func (orders Orderset) AddOrder(order simdriver.ClickEvent) {
	switch order.Type {
	case simdriver.Up:
		if order.Floor < 0 || order.Floor > simdriver.NumFloors-2 {
			fmt.Println("Attempted to add order to non-existing floor")
			return
		}
	case simdriver.Down:
		if order.Floor < 1 || order.Floor > simdriver.NumFloors-1 {
			fmt.Println("Attempted to add order to non-existing floor")
			return
		}
	case simdriver.Command:
		if order.Floor >= simdriver.NumFloors || order.Floor < 0 {
			fmt.Println("Attempted to add order to non-existing floor")
			return
		}
	}
	orders[order.Type][order.Floor] = true
	//simdriver.SetBtnLamp(order.Floor, order.Type, true)
}

func (orders Orderset) ClearOrders(floor int) {
	if floor != 0 {
		orders[simdriver.Down][floor] = false
	}
	if floor != (simdriver.NumFloors - 1) {
		orders[simdriver.Up][floor] = false
	}
	orders[simdriver.Command][floor] = false
}

func (state State) GetExpectedResponseTime(newOrder simdriver.ClickEvent) (responseTime float32) {
	if ((newOrder.Type == simdriver.Up) && (newOrder.Floor == simdriver.NumFloors-1)) || ((newOrder.Type == simdriver.Down) && (newOrder.Floor == 0)) {
		fmt.Println("Attempted to get response time of non-existing order type")
		responseTime = -1
		return
	}
	if state.Orders.IsOrder(newOrder) {
		fmt.Println("Order already exists")
		responseTime = -1
		return
	}

	//Initialize variables
	responseTime = 0

	currentOrders := state.Orders
	currentOrders.AddOrder(newOrder)
	currentDirection := state.Direction
	currentFloor := state.LastPassedFloor
	if state.Moving {
		if state.Direction == elevator.Up {
			currentFloor += 1
		} else {
			currentFloor -= 1
		}
		responseTime += movingPenalty
	} else if state.DoorOpen {
		responseTime += doorOpenPenalty
	}

	for {
		if currentOrders[simdriver.Command][currentFloor] || currentOrders.IsOrder(simdriver.ClickEvent{currentFloor, elevDirToDriverDir(currentDirection)}) {
			currentOrders.ClearOrders(currentFloor)
			if currentOrders.IsOrder(newOrder) {
				responseTime += stopTime
			} else {
				return
			}
		} else if currentOrders.IsOrderAhead(currentFloor, currentDirection) { //Ordre framover
			responseTime += floorTravelTime
			if currentDirection == elevator.Up {
				currentFloor += 1
			} else {
				currentFloor -= 1
			}
		} else if currentOrders.IsOrderBehind(currentFloor, currentDirection) || currentOrders.IsOrder(simdriver.ClickEvent{currentFloor, elevDirToDriverDir(currentDirection.OppositeDirection())}) { //Ordre bakover
			currentDirection = currentDirection.OppositeDirection()
		} else {
			//No orders left, to prevent erronous infinite loop this must be catched
			fmt.Println("Stuck forever ...")
		}
	}

	fmt.Println("Time estimator escaped for loop in invalid way")
	return
}

func elevDirToDriverDir(dir elevator.Direction_t) simdriver.BtnType {
	if dir == elevator.Up {
		return simdriver.Up
	} else {
		return simdriver.Down
	}
}

func DeepOrdersetCopy(from Orderset, to Orderset) {
	for btn, floorOrders := range from {
		for floor, isSet := range floorOrders {
			to[btn][floor] = isSet
		}
	}
}

func (oldState State) Diff(newState State) (lastPassedFloor, direction, moving, orders, valid, dooropen bool) {
	if oldState.LastPassedFloor == newState.LastPassedFloor {
		lastPassedFloor = true
	}
	if oldState.Direction == newState.Direction {
		direction = false
	}
	if oldState.Moving == newState.Moving {
		moving = false
	}
	if newState.Valid {
		valid = true
	}
	if oldState.DoorOpen == newState.DoorOpen {
		dooropen = true
	}

	orders = true
	for btn, floorOrders := range oldState.Orders {
		if _, ok := newState.Orders[btn]; !ok {
			orders = false
			return
		}
		for floor, isSet := range floorOrders {
			if newIsSet, ok := newState.Orders[btn][floor]; !ok || newIsSet != isSet {
				orders = false
				return
			}
		}
	}
	return
}
