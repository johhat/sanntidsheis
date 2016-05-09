package state

import (
	"../driver"
	"../elevator"
	"fmt"
	"os"
	"strconv"
)

type State struct {
	LastPassedFloor int
	Direction       elevator.Direction
	Moving          bool
	Orders          Orderset
	Valid           bool
	DoorOpen        bool
}

const (
	stopTime         float32 = 3
	floorPassingTime float32 = 0.385
	floorTravelTime  float32 = 2.2 + floorPassingTime
	movingPenalty    float32 = floorTravelTime / 2
	doorOpenPenalty  float32 = stopTime / 2
)

func (orders Orderset) RestoreInternalOrders() {
	for floor := 0; floor < driver.NumFloors; floor++ {
		file := "internalOrder" + strconv.Itoa(floor)
		if _, err := os.Stat(file); !os.IsNotExist(err) {
			orders[driver.Command][floor] = true
			driver.SetBtnLamp(floor, driver.Command, true)
			fmt.Println("Resored internal order at floor", floor)
		}
	}
}

func SaveInternalOrder(floor int) {
	_, err := os.Create("internalOrder" + strconv.Itoa(floor))
	if err != nil {
		fmt.Println(err)
	}
}

func DeleteSavedOrder(floor int) {
	file := "internalOrder" + strconv.Itoa(floor)
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		os.Remove(file)
	}
}

func (orders Orderset) IsOrder(event driver.ClickEvent) bool {
	if event.Floor < 0 || event.Floor > driver.NumFloors-1 {
		return false
	} else if event.Type == driver.Up && event.Floor == driver.NumFloors-1 {
		return false
	} else if event.Type == driver.Down && event.Floor == 0 {
		return false
	}
	return orders[event.Type][event.Floor]
}

func (orders Orderset) IsOrderAhead(currentFloor int, direction elevator.Direction) bool {
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

func (orders Orderset) IsOrderBehind(currentFloor int, direction elevator.Direction) bool {
	return orders.IsOrderAhead(currentFloor, direction.OppositeDirection())
}

func (orders Orderset) Init() {
	for floor := 0; floor < driver.NumFloors; floor++ {
		if floor == 0 {
			orders[driver.Up][floor] = false
			orders[driver.Command][floor] = false
		} else if floor == driver.NumFloors-1 {
			orders[driver.Down][floor] = false
			orders[driver.Command][floor] = false
		} else {
			orders[driver.Up][floor] = false
			orders[driver.Down][floor] = false
			orders[driver.Command][floor] = false
		}
	}
}

func (orders Orderset) AddOrder(order driver.ClickEvent) {
	switch order.Type {
	case driver.Up:
		if order.Floor < 0 || order.Floor > driver.NumFloors-2 {
			fmt.Println("Attempted to add order to non-existing floor")
			return
		}
	case driver.Down:
		if order.Floor < 1 || order.Floor > driver.NumFloors-1 {
			fmt.Println("Attempted to add order to non-existing floor")
			return
		}
	case driver.Command:
		if order.Floor >= driver.NumFloors || order.Floor < 0 {
			fmt.Println("Attempted to add order to non-existing floor")
			return
		}
	}
	orders[order.Type][order.Floor] = true
}

func (orders Orderset) ClearOrders(floor int) {
	if floor != 0 {
		orders[driver.Down][floor] = false
	}
	if floor != (driver.NumFloors - 1) {
		orders[driver.Up][floor] = false
	}
	orders[driver.Command][floor] = false
}

func (state State) GetExpectedResponseTime(newOrder driver.ClickEvent) (responseTime float32) {
	if ((newOrder.Type == driver.Up) && (newOrder.Floor == driver.NumFloors-1)) || ((newOrder.Type == driver.Down) && (newOrder.Floor == 0)) {
		fmt.Println("Attempted to get response time of non-existing order type")
		responseTime = -1
		return
	}
	if state.Orders.IsOrder(newOrder) {
		fmt.Println("Order already exists")
		responseTime = -1
		return
	}

	responseTime = 0
	currentOrders := state.CreateCopy().Orders
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

	fmt.Printf("\tResponse time sequence: ")
	if state.Moving {
		fmt.Printf("Move to next floor %v", currentFloor)
	}
	if state.DoorOpen {
		fmt.Printf("Close door")
	}

	for {
		if currentOrders[driver.Command][currentFloor] || currentOrders.IsOrder(driver.ClickEvent{currentFloor, elevDirToDriverDir(currentDirection)}) {
			currentOrders.ClearOrders(currentFloor)
			if currentOrders.IsOrder(newOrder) {
				responseTime += stopTime
				fmt.Printf("-> Stopping ")
			} else {
				fmt.Println("\n\tResponse time:", responseTime)
				return
			}
		} else if currentOrders.IsOrderAhead(currentFloor, currentDirection) { //Ordre framover
			responseTime += floorTravelTime
			if currentDirection == elevator.Up {
				currentFloor += 1
				fmt.Printf("-> Move up ")
			} else {
				currentFloor -= 1
				fmt.Printf("-> Move down ")
			}
		} else if currentOrders.IsOrderBehind(currentFloor, currentDirection) || currentOrders.IsOrder(driver.ClickEvent{currentFloor, elevDirToDriverDir(currentDirection.OppositeDirection())}) { //Ordre bakover
			currentDirection = currentDirection.OppositeDirection()
		} else {
			break
		}
	}

	fmt.Println("Time estimator escaped for loop in invalid way")
	return
}

func elevDirToDriverDir(dir elevator.Direction) driver.BtnType {
	if dir == elevator.Up {
		return driver.Up
	} else {
		return driver.Down
	}
}

func DeepOrdersetCopy(from Orderset, to Orderset) {
	for btn, floorOrders := range from {
		for floor, isSet := range floorOrders {
			to[btn][floor] = isSet
		}
	}
}

func (source State) CreateCopy() (stateCopy State) {
	stateCopy = State{
		source.LastPassedFloor,
		source.Direction,
		source.Moving,
		make(Orderset),
		source.Valid,
		source.DoorOpen}
	stateCopy.Orders[driver.Up] = make(FloorOrders)
	stateCopy.Orders[driver.Down] = make(FloorOrders)
	stateCopy.Orders[driver.Command] = make(FloorOrders)

	for btn, floorOrders := range source.Orders {
		for floor, isSet := range floorOrders {
			stateCopy.Orders[btn][floor] = isSet
		}
	}
	return
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
