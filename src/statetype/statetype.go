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
	floorTravelTime float32 = 2
	floorPassingTime float32 = 0.5
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

func (orders Orderset) isOrder(event simdriver.ClickEvent) bool {
	if event.Floor < 0 || event.Floor > simdriver.NumFloors-1 {
		fmt.Println("Attempted to check order for non-existing floor")
		return false
	} else if event.Type == simdriver.Up && event.Floor == simdriver.NumFloors-1 {
		fmt.Println("Attempted to check order for non-existing floor")
		return false
	} else if event.Type == simdriver.Down && event.Floor == 0 {
		fmt.Println("Attempted to check order for non-existing floor")
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

func (state State) GetExpectedResponseTime(newOrder simdriver.ClickEvent) (responseTime, bestCaseTime, worstCaseTime float32){
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
	/*
	Given that the order is assigned to this elevator
	- Find the time until the order is cleared by simulating the elevator behaviour
	- Then find best case and worst case time until the passenger will get to the 
	  destination floor, based on the other orders in the queue and the possible
	  destinations (max 3, min 1, average 2)
	*/

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
	var caseOrders Orderset

	for{
		if currentOrders[simdriver.Command][currentFloor] || currentOrders.isOrder(simdriver.ClickEvent{currentFloor, elevDirToDriverDir(currentDirection)}){
			fmt.Println("Stopping at floor",currentFloor)
			currentOrders.clearOrders(currentFloor)
			if currentOrders.isOrder(target){
				switch target.Floor{
				case bestCaseFloor:
					bestCaseTime += stopTime
				case worstCaseFloor:
					worstCaseTime += stopTime
				default:
					responseTime += stopTime
				}
			} else {
				switch target.Floor{
				case bestCaseFloor:
					target = simdriver.ClickEvent{worstCaseFloor, simdriver.Command}
					worstCaseTime += bestCaseTime
					caseOrders.addOrder(target)
					currentOrders = caseOrders
					fmt.Println("Response time:",responseTime,"bestCaseTime",bestCaseTime,"worstCaseTime",worstCaseTime)
				case worstCaseFloor:
					return
				default:
					target = simdriver.ClickEvent{bestCaseFloor, simdriver.Command}
					bestCaseTime += responseTime + stopTime
					caseOrders = currentOrders
					currentOrders.addOrder(target)
				}
			}
    	} else if currentOrders.isOrderAhead(currentFloor, currentDirection){ //Ordre framover
    		switch target.Floor{
			case bestCaseFloor:
				bestCaseTime += floorTravelTime
			case worstCaseFloor:
				worstCaseTime += floorTravelTime
			default:
				responseTime += floorTravelTime
			}
    		if currentDirection == elevator.Up{
    			currentFloor += 1
    		} else {
    			currentFloor -= 1
    		}
    	} else if currentOrders.isOrderBehind(currentFloor, currentDirection) || currentOrders.isOrder(simdriver.ClickEvent{currentFloor, elevDirToDriverDir(currentDirection.OppositeDirection())}){ //Ordre bakover
    		currentDirection = currentDirection.OppositeDirection()
    	} else {
    		//No orders left, to prevent erronous infinite loop this must be catched
    		fmt.Println("Stuck forever ...")
    	}
	}

	if bestCaseTime == 0 || worstCaseTime == 0{
		fmt.Println("getResponse time calculated invalid best case or worst case time")
	}
	return
}

func elevDirToDriverDir(dir elevator.Direction_t) simdriver.BtnType {
	if dir == elevator.Up{
		return simdriver.Up
	} else {
		return simdriver.Down
	}
	
}