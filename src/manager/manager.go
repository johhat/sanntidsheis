package manager

import (
	"fmt"
	"../simdriver"
	"../elevator"
	"../networking"
	"../statetype"
	"../com"
)

func Run(
	send_chan			chan<- messages.Message,
	receive_chan		<-chan messages.Message,
	connected_chan		<-chan string,
	disconnected_chan 	<-chan string,
	readDir_chan		<-chan readDirection,
	readOrder_chan		<-chan readOrder,
	completed_floor		<-chan int,
	clickEvent_chan		<-chan simdriver.ClickEvent,
	sensorEvent_chan	<-chan int,
	elev_error_chan		<-chan bool,
	//drop_conn_chan		chan<- bool,
	selfassign_chan		chan<- simdriver.ClickEvent){

	localIp := networking.GetLocalIp()

	//Initialize queue
	states := make(map[string]statetype.State)
	states[localIp] = State{-1, elevator.Up, false, make(statetype.Orderset), false}
	states[localIp].orders[simdriver.Up] = make(statetype.FloorOrders)
	states[localIp].orders[simdriver.Down] = make(statetype.FloorOrders)
	states[localIp].orders[simdriver.Command] = make(statetype.FloorOrders)
	states[localIp].orders.Init()

	for {
		select {
			case msg := <- receive_chan:
				switch msg.(type) {
				case com.OrderAssignmentMsg:
					if(msg.(com.OrderAssignmentMsg).Assignee != localIp){
						fmt.Println("Manager was assigned order with wrong IP address")
					} else {
						states[localIp].Orders.addOrder(msg.(com.OrderAssignment).Button)
						send_chan <- com.CreateClickEventMsg()//fix
					}
				case com.OrderEventMsg:
					//Sanity check av state-endring
					states[msg.(com.OrderEventMsg).Sender].Orders.addOrder(msg.(com.OrderEventMsg).Button)
				case com.SensorEventMsg:
					//Sanity check av state-endring
					state[msg.(com.SensorEventMsg).Sender].LastPassedFloor = msg.(com.SensorEventMsg).NewState.LastPassedFloor
					state[msg.(com.SensorEventMsg).Sender].Direction = msg.(com.SensorEventMsg).NewState.Direction
					state[msg.(com.SensorEventMsg).Sender].Moving = msg.(com.SensorEventMsg).NewState.Moving
					state[msg.(com.SensorEventMsg).Sender].SequenceNumber = msg.(com.SensorEventMsg).NewState.SequenceNumber
					state[msg.(com.SensorEventMsg).Sender].DoorOpen = msg.(com.SensorEventMsg).NewState.DoorOpen
				case com.InitialStateMsg:
					//Sanity check av state-endring
					state[msg.(com.InitialStateMsg).Sender].LastPassedFloor = msg.(com.InitialStateMsg).NewState.LastPassedFloor
					state[msg.(com.InitialStateMsg).Sender].Direction = msg.(com.InitialStateMsg).NewState.Direction
					state[msg.(com.InitialStateMsg).Sender].Moving = msg.(com.InitialStateMsg).NewState.Moving
					state[msg.(com.InitialStateMsg).Sender].SequenceNumber = msg.(com.InitialStateMsg).NewState.SequenceNumber
					state[msg.(com.InitialStateMsg).Sender].DoorOpen = msg.(com.InitialStateMsg).NewState.DoorOpen
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

			case completed := <-completed_floor:
				states[localIp].Orders.clearOrders(completed.floor)

			case disconnected := <-disconnected_chan:

				// We should redistribute if we have the highest IP address, check if we do
				shouldRedistribute := true
				for ip, _ := range states {
					remoteIpHighest, err := networking.HasHighestIp(ip,localIp)
					if err != nil {
						fmt.Println(err)
					}
					if remoteIsHighest && (ip != disconnected) {
						shouldRedistribute = false
						break
					}
				}

				if shouldRedistribute {
					// For every external order that needs to be redistributed
					for btnType, floorOrders := range states[disconnected].Orders { 
						if btnType != simdriver.Command{
							for floor, isSet := range floorOrders {
								if isSet {
									// Find the elevator with shortest expected response time
									bestIp := localIp // Local elevator is default
									shortestResponseTime := 99999.9 //Inf
									for ip, state := range states{
										if time := state.GetExpectedResponseTime(simdriver{floor,btnType}); time < shortestResponseTime {
											shortestResponseTime = time
											bestIp = ip
										}
									}
									// Send order to the best elevator
									fmt.Println("Reassigning order floor",floor,",type",btnType,"to ip",bestIp)
									if bestIp == localIp{
										states[localIp].Sequencenumber += 1
										selfassign_chan <- simdriver{floor,btnType}
										send_chan <- com.CreateClickEventMsg()//fix
									} else {
										send_chan <- com.CreateOrderAssignmentMsg()//fix
									}
								}
							}
						}
					}
				}
				delete(states,ip)

			case ip := <-newElevator:
				states[ip] = statetype.State{-1, elevator.Up, false, make(statetype.Orderset), false}
				states[ip].orders[simdriver.Up] = make(statetype.FloorOrders)
				states[ip].orders[simdriver.Down] = make(statetype.FloorOrders)
				states[ip].orders[simdriver.Command] = make(statetype.FloorOrders)
				send_chan <- com.CreateInitialStateMsg()//fix
			case buttonClick := clickEvent_chan: //Må endre til å inkludere stoppknapp
				//Hvis stoppknapp
					//Hvis vi er i service-modus
						//Init på nytt?
						//Restart heartbeats
						//Lytt til heartbeats
				if(buttonClick.Type == simdriver.Command){
					if !state[localIp].Orders.isOrder(buttonClick){
						selfassign_chan <- buttonClick
						saveInternalOrder(buttonClick.Floor)
						states[localIp].Sequencenumber += 1
						send_chan <- com.CreateClickEventMsg()//fix
					}
				} else {
					for _, state := range states {
						if state.Orders.isOrder(buttonClick) {
							break //Order already exists
						}
					}
					bestIp := localIp // Local elevator is default
					shortestResponseTime := 99999.9 //Inf
					for ip, state := range states{
						if time := state.GetExpectedResponseTime(simdriver{buttonClick.Floor,buttonClick.Type}); time < shortestResponseTime {
							shortestResponseTime = time
							bestIp = ip
						}
					}
					if bestIp == localIp{
						selfassign_chan <- simdriver{floor,btnType}
						states[localIp].Sequencenumber += 1
						send_chan <- com.CreateClickEventMsg()//fix
					} else {
						send_chan <- com.CreateOrderAssignmentMsg()//fix
					}

				}
			case sensorEvent := <-sensorEvent_chan:
				// Oppdater state, inkrementer sekvensnummer
				//
				com.CreateSensorEventMsg()//fix
			case <-elev_error_chan:
				// disconnect TCP
		}
	}
}

func sanityCheck(oldState statetype.State, newState statetype.State, event com.EventType) bool {
	//Check sequence number
	if newState.Sequencenumber != (oldState.Sequencenumber + 1) {
		fmt.Println("Received state message with out of order sequence number. Old:",oldState.Sequencenumber,"New:",newState.Sequencenumber)
		if newState.Sequencenumber > (oldState.Sequencenumber + 1){
			//Skipped message
		} else if newState.Sequencenumber < (oldState.Sequencenumber + 1){
			//Received skipped message
		}
	}

	//Check if old state + event = new state
	lastPassedFloorEqual, directionEqual, movingEqual, ordersEqual, validEqual, dooropenEqual := oldState.diff(newState)
	switch event {
	case com.NewExternalOrder:
		if(lastPassedFloorEqual && directionEqual && movingEqual && !ordersEqual && validEqual && dooropenEqual){
			//Sjekk av bare én ordre er endret og at den er ekstern
		}
		return false
	case com.NewInternalOrder:
		if(lastPassedFloorEqual && directionEqual && movingEqual && !ordersEqual && validEqual && dooropenEqual){
			//Sjekk av bare én ordre er endret og at den er intern
		}
		return false
	case com.PassingFloor:
		if(!lastPassedFloorEqual && directionEqual && movingEqual && ordersEqual && validEqual && dooropenEqual){
			switch newState.Direction {
			case elevator.Up:
				if newState.LastPassedFloor == (oldState.LastPassedFloor + 1) {
					return true
				}
			case elevator.Down:
				if newState.LastPassedFloor == (oldState.LastPassedFloor - 1) {
					return true
				}
			}			
		}
		return false
	case com.DoorOpenedByInternalOrder:
		if(lastPassedFloorEqual && directionEqual && movingEqual && ordersEqual && validEqual && !dooropenEqual){
			if newState.DoorOpen {
				return true
			}
		}
		return false
	case com.StoppingToFinishOrder:
	case com.LeavingFloor:
}
