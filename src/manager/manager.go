package manager

import (
	"fmt"
	"../simdriver"
	"time"
	"../elevator"
	"../networking"
	."../com"
	"os"
)



const(
	stopTime float32 = 3
	floorTravelTime float32 = 2
	floorPassingTime float32 = 0.5
	movingPenalty float32 = floorTravelTime/2
	doorOpenPenalty float32 = doorOpenTime/2
)

func (orders Orderset) getExpectedResponseTime(simdriver.ClickEvent, ip string) (responseTime, bestCaseTime, worstCaseTime float32){
	if (direction == Up && floor == NumFloors-1) || (direction == Down && floor == 0){
		fmt.Println("Attempted to get response time of non-existing order type")
		return (-1, -1, -1)
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
	if direction == Up {
		worstCaseFloor = simdriver.NumFloors-1
		bestCaseFloor = currentFloor+1
	} else {
		worstCaseFloor = 0
		bestCaseFloor = currentFloor-1
	}
	
	//Initialize variables
	responseTime := 0
	bestCaseTime := 0
	worstCaseTime := 0

	currentOrders := orders[id]
	orders[id].append(targetFloor, direction)
	currentDirection := direction[id]
	currentFloor := elevator[id]

	if targetFloor == currentFloor && direction == currentDirection{
		responseTime = 0
	}

	for{
		if internalQueue[last_passed_floor] || isOrder(last_passed_floor, current_direction){
			//Clear orders on floor
			if !orderCleared(){
				wtime += stopTime
			}
    	} else if isOrderAhead(last_passed_floor, current_direction){ //Ordre framover
    		wtime += floorTravelTime
    		currentFloor = nextFloor
    	} else if isOrderBehind(last_passed_floor, current_direction) || isOrder(last_passed_floor, oppositeDirection(current_direction)){ //Ordre bakover
    		current_direction = oppositeDirection(current_direction)
    	} else {
    		//No orders left, to prevent erronous infinite loop this must be catched
    	}
	}

	if bestCaseTime == 0 || worstCaseTime == 0{
		//Handle error
	}
}


func Run(
	send_chan			chan<- messages.Message
	receive_chan		<-chan messages.Message
	connected_chan		<-chan string
	disconnected_chan 	<-chan string
	readDir_chan		<-chan readDirection
	readOrder_chan		<-chan readOrder
	clearOrder_chan 	<-chan deleteOp
	clickEvent_chan		<-chan simdriver.ClickEvent
	sensorEvent_chan	<-chan int
	elev_error_chan		<-chan bool
	selfassign_chan		chan<- simdriver.ClickEvent
	) {

	localIp := networking.GetLocalIp()

	//Initialize queue
	states := make(map[string]State)
	states[localIp] = state{-1, elevator.Up, false, make(Orders), false}
	states[localIp].orders[simdriver.Up] = make(FloorOrders)
	states[localIp].orders[simdriver.Down] = make(FloorOrders)
	states[localIp].orders[simdriver.Command] = make(FloorOrders)
	states[localIp].orders.Init()

	for {
		select {
			case msg := <- receive_chan:
				switch msg.(type) {
				case com.OrderAssignmentMsg:
					if(msg.(com.OrderAssignment).Assignee != localIp){
						fmt.Println("Manager was assigned order with wrong IP address")
					} else {
						states[localIp].Orders.addOrder(msg.(com.OrderAssignment).Button)
					}
				case com.ClickEventMsg:
					//Sanity check av state-endring
					msg.(com.AssignOrder).
				case com.SensorEventMsg:
					//
				case com.InitialStateMsg:
					//
				default:
					fmt.Println("Manager received invalid message")
				}
			case readOrder := <-readOrder_chan:
				readOrder.resp <- states[localIp].Orders.isOrder(readOrder.order)
			case readDir := <-readDir_chan:
				switch(readDir.request){
            	case isOrderAhead:
            		readDir.resp <- states[localIp].Orders.isOrderAhead(readDir.floor, readDir.direction)
            	case isOrderBehind:
            		readDir.resp <- states[localIp].Orders.isOrderBehind(readDir.floor, readDir.direction)
            	}
			case clearOrder := <-clearOrder_chan:
				states[localIp].Orders.clearOrders(clearOrder.floor)
                clearOrder.resp <- true
			case disconnected := <-disconnected_chan:
				for ip, _ := range states {
					if ip > localIp && ip != disconnected{
						//Fjern heisen fra states
						break
					}
				}
				//Vi refordeler alle eksterne ordre
					//Sjekk forventet responstid for alle heiser, velg den beste for hver ordre
				//Fjern heisen fra states
				for 
			case ip := <-newElevator:
				states[ip] = state{-1, elevator.Up, false, make(Orders), false}
				states[ip].orders[simdriver.Up] = make(FloorOrders)
				states[ip].orders[simdriver.Down] = make(FloorOrders)
				states[ip].orders[simdriver.Command] = make(FloorOrders)
				send_chan<-com.Event{com.SendState, }
			case buttonClick := clickEvent_chan: //Må endre til å inkludere stoppknapp
				//Hvis stoppknapp
					//Hvis vi er i service-modus
						//Init på nytt?
						//Restart heartbeats
						//Lytt til heartbeats
				for _, state := range states {
					if state.Orders.isOrder(buttonClick) {
						break
					}
				}
				if(buttonClick.Type == simdriver.Command){
					selfassign_chan <- buttonClick
					//Lagre til disk
					send_chan <- //Varsle om ny state
				} else {
					//Sjekk forventet responstid for alle heiser, velg den beste
				}
			case sensorEvent := <-sensorEvent_chan:
				//
			//case assignedOrder:
				//legg til ordre lokalt
			//case localOrderFinished:
			case <-elev_error_chan:
				// disconnect TCP
			//case remoteOrderFinished:
				//Sjekk om mottatt ordreliste samsvarer med gammel-fullført ordre
				//Oppdater oversikt over den andre heisens ordre
		}
	}
}

func saveInternalOrder(floor int){
	f, err := os.Create("/internalOrder"+floor)
}

func (orders *Orders) restoreInternalOrders(){
	for floor := 0; floor < simdriver.NumFloors; floor++ {
		if _, err := os.Stat("/internalOrder"+floor); !os.IsNotExist(err) {
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

func (orders Orderset) isOrderAhead(currentFloor int, direction Direction_t) bool {
	for _, buttonOrders := range orders{
		for floor, isSet := range buttonOrders {
			if direction == Up && floor > currentFloor && isSet {
				return true
			} else if direction == Down && floor < currentFloor && isSet {
				return true
			}
		}
	}
	return false
}

func (orders Orderset) isOrderBehind(currentFloor int, direction Direction_t) bool{
	return orders.isOrderAhead(currentFloor, direction.oppositeDirection())
}

func (orders *Orderset) Init(){
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

func (orders *Orderset) addOrder(order simdriver.ClickEvent){
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

func sanityCheck(oldState State, newState State, event Event) bool {
	//Check if old state + event = new state
	switch event {
	case com.NewExternalOrder:
		//
	case com.NewInternalOrder:
		//
	case SelfAssignedOrder:
		//
	case PassingFloor:
		//
	case DoorOpenedByInternalOrder:
		//
	case StoppingToFinishOrder:
		//
	case LeavingFloor:
		//


	}




}