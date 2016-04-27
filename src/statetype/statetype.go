package statetype

import(
	"../simdriver"
	"../elevator"
	"os"
	"strconv"
	"fmt"
)

type FloorOrders map[int]bool
type Orderset map[simdriver.BtnType]FloorOrders

type State struct {
	LastPassedFloor int
	Direction       elevator.Direction_t
	Moving          bool
	Orders          Orderset
	Valid           bool
	SequenceNumber  int
	DoorOpen        bool
}

const(
	stopTime float32 = 3
	floorTravelTime float32 = 2//2.2+floorPassingTime
	floorPassingTime float32 = 0.385
	movingPenalty float32 = floorTravelTime/2
	doorOpenPenalty float32 = stopTime/2
)

func (orders Orderset) restoreInternalOrders(){
	for floor := 0; floor < simdriver.NumFloors; floor++ {
		if _, err := os.Stat("/internalOrder"+strconv.Itoa(floor)); !os.IsNotExist(err) {
  			orders[simdriver.Command][floor] = true
		}
	}
}

func saveInternalOrder(floor int){
	f, err := os.Create("/internalOrder"+strconv.Itoa(floor))
}

func (orders Orderset) isOrder(event simdriver.ClickEvent) bool {
	if event.Floor < 0 || event.Floor > simdriver.NumFloors-1 {
		fmt.Println("Attempted to check order for non-existing floor")
		return false
	} else if event.Type == simdriver.Up && event.Floor == simdriver.NumFloors-1 {
		return false
	} else if event.Type == simdriver.Down && event.Floor == 0 {
		return false
	}
	return orders[event.Type][event.Floor]
}

func (orders Orderset) isOrderAhead(currentFloor int, direction elevator.Direction_t) bool {
	for _, buttonOrders := range orders{
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

func (orders Orderset) isOrderBehind(currentFloor int, direction elevator.Direction_t) bool{
	return orders.isOrderAhead(currentFloor, direction.OppositeDirection())
}

func (orders Orderset) Init(){
	for floor := 0; floor < simdriver.NumFloors; floor++ {
		if floor == 0{
			orders[simdriver.Up][floor] = false
			orders[simdriver.Command][floor] = false
		} else if floor == simdriver.NumFloors - 1 {
			orders[simdriver.Down][floor] = false
			orders[simdriver.Command][floor] = false
		} else {
			orders[simdriver.Up][floor] = false
			orders[simdriver.Down][floor] = false
			orders[simdriver.Command][floor] = false
		}
	}
}

func (orders Orderset) addOrder(order simdriver.ClickEvent){
	switch(order.Type){
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

func (orders Orderset) clearOrders(floor int){
	if floor != 0 {
		orders[simdriver.Down][floor] = false
	}
	if floor != (simdriver.NumFloors-1) {
		orders[simdriver.Up][floor] = false
	}
	orders[simdriver.Command][floor] = false
}

func (state State) GetExpectedResponseTime(newOrder simdriver.ClickEvent) (responseTime float32){
	fmt.Println("Estimating times for new order on floor",newOrder.Floor,"of type",newOrder.Type)
	if ((newOrder.Type == simdriver.Up) && (newOrder.Floor == simdriver.NumFloors-1)) || ((newOrder.Type == simdriver.Down) && (newOrder.Floor == 0)){
		fmt.Println("Attempted to get response time of non-existing order type")
		responseTime = -1
		bestCaseTime = -1
		worstCaseTime = -1
		return
	}
	if state.Orders.isOrder(newOrder){
		fmt.Println("Order already exists")
		responseTime = -1
		bestCaseTime = -1
		worstCaseTime = -1
		return
	}

	//Among the possible destinations, find the floors closest and furthest away
	var bestCaseFloor int
	var worstCaseFloor int
	if newOrder.Type == simdriver.Up {
		worstCaseFloor = simdriver.NumFloors-1
		bestCaseFloor = newOrder.Floor+1
	} else {
		worstCaseFloor = 0
		bestCaseFloor = newOrder.Floor-1
	}
	fmt.Println("Best case floor:",bestCaseFloor,"Worst case floor:",worstCaseFloor)
	
	//Initialize variables
	responseTime = 0
	bestCaseTime = 0
	worstCaseTime = 0

	currentOrders := state.Orders
	currentOrders.addOrder(newOrder)
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
	target := newOrder
	caseOrders := make(Orderset)
	caseOrders[simdriver.Down] = make(FloorOrders)
	caseOrders[simdriver.Up] = make(FloorOrders)
	caseOrders[simdriver.Command] = make(FloorOrders)

	for{
		if currentOrders[simdriver.Command][currentFloor] || currentOrders.isOrder(simdriver.ClickEvent{currentFloor, elevDirToDriverDir(currentDirection)}){
			//fmt.Println("Stopping at floor",currentFloor)
			currentOrders.clearOrders(currentFloor)
			if currentOrders.isOrder(target){
				switch target.Floor{
				case bestCaseFloor:
					bestCaseTime += stopTime
					//fmt.Println("Stop: Bestcasetime + 3")
				case worstCaseFloor:
					worstCaseTime += stopTime
					//fmt.Println("Stop: Worstcasetime + 3")
				default:
					responseTime += stopTime
					//fmt.Println("Stop: Responsetime + 3")
				}
			} else {
				switch target.Floor{
				case bestCaseFloor:
					target = simdriver.ClickEvent{worstCaseFloor, simdriver.Command}
					worstCaseTime += responseTime + stopTime
					//fmt.Println("Worstcasetime + responsetime + 3")
					caseOrders.addOrder(target)
					deepOrdersetCopy(caseOrders,currentOrders)
					currentFloor = newOrder.Floor
				case worstCaseFloor:
					return
				default:
					target = simdriver.ClickEvent{bestCaseFloor, simdriver.Command}
					bestCaseTime += responseTime + stopTime
					//fmt.Println("Bestcasetime + responsetime + 3")
					deepOrdersetCopy(currentOrders,caseOrders)
					currentOrders.addOrder(target)
				}
			}
    	} else if currentOrders.isOrderAhead(currentFloor, currentDirection){ //Ordre framover
    		switch target.Floor{
			case bestCaseFloor:
				bestCaseTime += floorTravelTime
				//fmt.Println("Move: Bestcasetime + 2")
			case worstCaseFloor:
				worstCaseTime += floorTravelTime
				//fmt.Println("Move: Worstcasetime + 2")
			default:
				responseTime += floorTravelTime
				//fmt.Println("Move: Responsetime + 2")
			}
    		if currentDirection == elevator.Up{
    			currentFloor += 1
    			//fmt.Println("Going up to",currentFloor)
    		} else {
    			currentFloor -= 1
    			//fmt.Println("Going down to",currentFloor)
    		}
    	} else if currentOrders.isOrderBehind(currentFloor, currentDirection) || currentOrders.isOrder(simdriver.ClickEvent{currentFloor, elevDirToDriverDir(currentDirection.OppositeDirection())}){ //Ordre bakover
    		currentDirection = currentDirection.OppositeDirection()
    		//fmt.Println("Turning around")
    	} else {
    		//No orders left, to prevent erronous infinite loop this must be catched
    		//fmt.Println("Stuck forever ...")
    	}
	}

	fmt.Println("Time estimator escaped for loop in invalid way")
	return
}

func elevDirToDriverDir(dir elevator.Direction_t) simdriver.BtnType {
	if dir == elevator.Up{
		return simdriver.Up
	} else {
		return simdriver.Down
	}
}

func deepOrdersetCopy(from Orderset, to Orderset) {
	for btn,floorOrders := range from{
		for floor, isSet := range floorOrders{
			to[btn][floor] = isSet
		}
	}
}

func (oldState State) diff(newState State) (lastPassedFloor, direction, moving, orders, valid, dooropen bool) {
	if oldState.LastPassedFloor == newState.LastPassedFloor{
		lastPassedFloor = true
	}
	if oldState.Direction == newState.Direction {
		direction = false
	}
	if oldState.Moving == newState.Moving{
		moving = false
	}
	if newState.Valid{
		valid = true
	}
	if oldState.DoorOpen == newState.DoorOpen{
		dooropen = true
	}

	orders = true
	for btn, floorOrders := range oldState.Orders {
		if _, ok := newState[btn]; !ok {
			orders = false
			return
		}
		for floor, isSet := range floorOrders{
			if newIsSet, ok := newState[btn][floor]; !ok || newIsSet != isSet {
				orders = false
				return
			}
		}
	}	
}