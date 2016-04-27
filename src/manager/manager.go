package manager

import (
	"../com"
	"../elevator"
	"../networking"
	"../simdriver"
	"../statetype"
	"fmt"
)

func Run(
	send_chan chan<- com.Message,
	receive_chan <-chan com.Message,
	connected_chan <-chan string,
	disconnected_chan <-chan string,
	readDir_chan <-chan elevator.ReadDirection,
	readOrder_chan <-chan elevator.ReadOrder,
	completed_floor <-chan int,
	door_closed_chan <-chan bool,
	clickEvent_chan <-chan simdriver.ClickEvent,
	sensorEvent_chan <-chan int,
	floor_reached chan<- int,
	start_moving <-chan bool,
	PassingFloor <-chan bool,
	elev_error_chan chan bool,
	disconnectFromNetwork chan<- bool,
	reconnectToNetwork chan<- bool,
	networking_timeout <-chan bool) {

	localIp := networking.GetLocalIp()
	error_state := false

	//Initialize queue
	states := make(map[string]statetype.State)
	states[localIp] = statetype.State{-1, elevator.GetCurrentDirection(), false, make(statetype.Orderset), false, 0, false}
	states[localIp].Orders[simdriver.Up] = make(statetype.FloorOrders)
	states[localIp].Orders[simdriver.Down] = make(statetype.FloorOrders)
	states[localIp].Orders[simdriver.Command] = make(statetype.FloorOrders)
	states[localIp].Orders.Init()

	for {
		select {
		case msg := <-receive_chan:
			switch msg.(type) {
			case com.OrderAssignmentMsg:
				if msg.(com.OrderAssignmentMsg).Assignee != localIp {
					fmt.Println("Manager was assigned order with wrong IP address")
				} else {
					states[localIp].Orders.AddOrder(msg.(com.OrderAssignmentMsg).Button)
					send_chan <- com.OrderEventMsg{msg.(com.OrderAssignmentMsg).Button, states[localIp], localIp}
				}
			case com.OrderEventMsg:
				//Sanity check av state-endring
				states[msg.(com.OrderEventMsg).Sender].Orders.AddOrder(msg.(com.OrderEventMsg).Button)
				//Skru på knappelys hvis ordren er ekstern
			case com.SensorEventMsg:
				//Sanity check av state-endring
				tmp := states[msg.(com.SensorEventMsg).Sender]
				tmp.Direction = msg.(com.SensorEventMsg).NewState.Direction
				tmp.LastPassedFloor = msg.(com.SensorEventMsg).NewState.LastPassedFloor
				tmp.Moving = msg.(com.SensorEventMsg).NewState.Moving
				tmp.SequenceNumber = msg.(com.SensorEventMsg).NewState.SequenceNumber
				tmp.DoorOpen = msg.(com.SensorEventMsg).NewState.DoorOpen
				states[msg.(com.SensorEventMsg).Sender] = tmp
			case com.InitialStateMsg:
				//Sanity check av state-endring
				tmp := states[msg.(com.InitialStateMsg).Sender]
				tmp.LastPassedFloor = msg.(com.InitialStateMsg).NewState.LastPassedFloor
				tmp.Direction = msg.(com.InitialStateMsg).NewState.Direction
				tmp.Moving = msg.(com.InitialStateMsg).NewState.Moving
				tmp.SequenceNumber = msg.(com.InitialStateMsg).NewState.SequenceNumber
				tmp.DoorOpen = msg.(com.InitialStateMsg).NewState.DoorOpen
				tmp.Valid = true
				states[msg.(com.InitialStateMsg).Sender] = tmp
				statetype.DeepOrdersetCopy(msg.(com.InitialStateMsg).NewState.Orders, states[msg.(com.InitialStateMsg).Sender].Orders)
			default:
				fmt.Println("Manager received invalid message")
			}
		case readOrder := <-readOrder_chan:
			readOrder.Resp <- states[localIp].Orders.IsOrder(readOrder.Order)
		case readDir := <-readDir_chan:
			switch readDir.Request {
			case elevator.IsOrderAhead:
				readDir.Resp <- states[localIp].Orders.IsOrderAhead(readDir.Floor, readDir.Direction)
			case elevator.IsOrderBehind:
				readDir.Resp <- states[localIp].Orders.IsOrderBehind(readDir.Floor, readDir.Direction)
			}

		case completed := <-completed_floor:
			tmp := states[localIp]
			tmp.Moving = false
			tmp.DoorOpen = true
			tmp.Orders.ClearOrders(completed)
			simdriver.SetBtnLamp(completed, simdriver.Up, false)
			simdriver.SetBtnLamp(completed, simdriver.Down, false)
			simdriver.SetBtnLamp(completed, simdriver.Command, false)
			states[localIp] = tmp
			send_chan <- com.SensorEventMsg{com.StoppingToFinishOrder, states[localIp], localIp}
		case <-door_closed_chan:
			tmp := states[localIp]
			tmp.DoorOpen = false
			states[localIp] = tmp
			send_chan <- com.SensorEventMsg{com.DoorClosed, states[localIp], localIp}
		case connected := <-connected_chan:
			states[connected] = statetype.State{-1, elevator.Up, false, make(statetype.Orderset), false, 0, false}
			states[connected].Orders[simdriver.Up] = make(statetype.FloorOrders)
			states[connected].Orders[simdriver.Down] = make(statetype.FloorOrders)
			states[connected].Orders[simdriver.Command] = make(statetype.FloorOrders)
			send_chan <- com.InitialStateMsg{states[localIp], localIp}

		case disconnected := <-disconnected_chan:

			// We should redistribute if we have the highest IP address, check if we do
			shouldRedistribute := true
			for ip, _ := range states {
				remoteIpHighest, err := networking.HasHighestIp(ip, localIp)
				if err != nil {
					fmt.Println(err)
				}
				if remoteIpHighest && (ip != disconnected) {
					shouldRedistribute = false
					break
				}
			}

			if shouldRedistribute {
				// For every external order that needs to be redistributed
				for btnType, floorOrders := range states[disconnected].Orders {
					if btnType != simdriver.Command {
						for floor, isSet := range floorOrders {
							if isSet {
								// Find the elevator with shortest expected response time
								bestIp := localIp                          // Local elevator is default
								var shortestResponseTime float32 = 99999.9 //Inf
								for ip, state := range states {
									if time := state.GetExpectedResponseTime(simdriver.ClickEvent{floor, btnType}); time < shortestResponseTime {
										shortestResponseTime = time
										bestIp = ip
									}
								}
								// Send order to the best elevator
								fmt.Println("Reassigning order floor", floor, ",type", btnType, "to ip", bestIp)
								if bestIp == localIp {
									tmp := states[localIp]
									tmp.SequenceNumber += 1
									tmp.Orders.AddOrder(simdriver.ClickEvent{floor, btnType})
									states[localIp] = tmp
									send_chan <- com.OrderEventMsg{simdriver.ClickEvent{floor, btnType}, states[localIp], localIp}
								} else {
									send_chan <- com.OrderAssignmentMsg{simdriver.ClickEvent{floor, btnType}, bestIp}
								}
							}
						}
					}
				}
			}
			delete(states, disconnected)

		case buttonClick := <-clickEvent_chan: //Må endre til å inkludere stoppknapp
			//Hvis stoppknapp {
			if error_state {
				//Init på nytt?
				//Restart heartbeats
				//Lytt til heartbeats
				//error_state = false
			}
			//continue
			//}
			if buttonClick.Type == simdriver.Command {
				if !states[localIp].Orders.IsOrder(buttonClick) {
					states[localIp].Orders.AddOrder(buttonClick)
					simdriver.SetBtnLamp(buttonClick.Floor, buttonClick.Type, true)
					//statetype.SaveInternalOrder(buttonClick.Floor)
					//Add to local state
					tmp := states[localIp]
					tmp.SequenceNumber += 1
					states[localIp] = tmp
					send_chan <- com.OrderEventMsg{buttonClick, states[localIp], localIp}
				}
			} else {
				for _, state := range states {
					if state.Orders.IsOrder(buttonClick) {
						fmt.Println("Order already exists:", buttonClick)
						break //Order already exists
					}
				}
				bestIp := localIp                          // Local elevator is default
				var shortestResponseTime float32 = 99999.9 //Inf
				for ip, state := range states {
					if time := state.GetExpectedResponseTime(buttonClick); time < shortestResponseTime {
						shortestResponseTime = time
						bestIp = ip
					}
				}
				fmt.Println("Best IP for order", buttonClick, "is", bestIp)
				if bestIp == localIp {
					tmp := states[localIp]
					tmp.SequenceNumber += 1
					tmp.Orders.AddOrder(buttonClick)
					states[localIp] = tmp
					send_chan <- com.OrderEventMsg{buttonClick, states[localIp], localIp}
				} else {
					send_chan <- com.OrderAssignmentMsg{buttonClick, bestIp}
				}
				simdriver.SetBtnLamp(buttonClick.Floor, buttonClick.Type, true)

			}
		case sensorEvent := <-sensorEvent_chan:
			if sensorEvent == -1 && !states[localIp].Moving {
				elev_error_chan <- true
				continue
			}
			if sensorEvent == -1 {
				tmp := states[localIp]
				tmp.SequenceNumber += 1
				states[localIp] = tmp
				send_chan <- com.SensorEventMsg{com.LeavingFloor, states[localIp], localIp}
			} else {
				tmp := states[localIp]
				tmp.LastPassedFloor = sensorEvent
				if !tmp.Valid {
					tmp.Valid = true
				} else {
					floor_reached <- sensorEvent
				}
				states[localIp] = tmp

			}
		case <-start_moving:
			tmp := states[localIp]
			tmp.Moving = true
			states[localIp] = tmp
		case <-PassingFloor:
			send_chan <- com.SensorEventMsg{com.PassingFloor, states[localIp], localIp}
		case <-elev_error_chan:
			error_state = true
			disconnectFromNetwork <- true
			//Stop light on
			//Remove remote elevators from states?
		}
	}
}

func sanityCheck(oldState statetype.State, newState statetype.State, event com.EventType) bool {
	//Check sequence number
	if newState.SequenceNumber != (oldState.SequenceNumber + 1) {
		fmt.Println("Received state message with out of order sequence number. Old:", oldState.SequenceNumber, "New:", newState.SequenceNumber)
		if newState.SequenceNumber > (oldState.SequenceNumber + 1) {
			//Skipped message
		} else if newState.SequenceNumber < (oldState.SequenceNumber + 1) {
			//Received skipped message
		}
	}

	//Check if old state + event = new state
	lastPassedFloorEqual, directionEqual, movingEqual, ordersEqual, validEqual, dooropenEqual := oldState.Diff(newState)
	switch event {
	//case com.NewExternalOrder:
	//	if(lastPassedFloorEqual && directionEqual && movingEqual && !ordersEqual && validEqual && dooropenEqual){
	//		//Sjekk av bare én ordre er endret og at den er ekstern
	//	}
	//	return false
	//case com.NewInternalOrder:
	//	if(lastPassedFloorEqual && directionEqual && movingEqual && !ordersEqual && validEqual && dooropenEqual){
	//		//Sjekk av bare én ordre er endret og at den er intern
	//	}
	//	return false
	case com.PassingFloor:
		if !lastPassedFloorEqual && directionEqual && movingEqual && ordersEqual && validEqual && dooropenEqual {
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
		if lastPassedFloorEqual && directionEqual && movingEqual && ordersEqual && validEqual && !dooropenEqual {
			if newState.DoorOpen {
				return true
			}
		}
		return false
	case com.StoppingToFinishOrder:
	case com.LeavingFloor:
	}
	return false
}
