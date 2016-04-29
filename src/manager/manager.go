package manager

import (
	"../com"
	driver "../driver"
	"../elevator"
	"../networking"
	"../statetype"
	"fmt"
)

type unconfirmedOrder struct {
	Button   driver.ClickEvent
	Reciever string
}

func Run(
	send_chan chan<- com.Message,
	receive_chan <-chan com.Message,
	connected_chan <-chan string,
	disconnected_chan <-chan string,
	readDir_chan <-chan elevator.ReadDirection,
	readOrder_chan <-chan elevator.ReadOrder,
	completed_floor <-chan int,
	door_closed_chan <-chan bool,
	clickEvent_chan chan driver.ClickEvent,
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

	unconfirmedOrders := make(map[unconfirmedOrder]bool)

	//Initialize queue
	states := make(map[string]statetype.State)
	states[localIp] = statetype.State{-1, elevator.GetCurrentDirection(), false, make(statetype.Orderset), false, 0, false}
	states[localIp].Orders[driver.Up] = make(statetype.FloorOrders)
	states[localIp].Orders[driver.Down] = make(statetype.FloorOrders)
	states[localIp].Orders[driver.Command] = make(statetype.FloorOrders)
	states[localIp].Orders.Init()

	for {
		select {
		case msg := <-receive_chan:
			switch msg := msg.(type) {
			case com.OrderAssignmentMsg:
				if msg.Assignee != localIp {
					fmt.Println("Manager was assigned order with wrong IP address")
					break
				}
				fmt.Println("Received assignment:", msg.Button)
				states[localIp].Orders.AddOrder(msg.Button)
				driver.SetBtnLamp(msg.Button.Floor, msg.Button.Type, true)
				send_chan <- com.OrderEventMsg{msg.Button, states[localIp].CreateCopy(), localIp}

			case com.OrderEventMsg:
				//Sanity check av state-endring
				delete(unconfirmedOrders, unconfirmedOrder{msg.Button, msg.Sender})
				states[msg.Sender].Orders.AddOrder(msg.Button)
				if msg.Button.Type != driver.Command {
					driver.SetBtnLamp(msg.Button.Floor, msg.Button.Type, true)
				}
			case com.SensorEventMsg:
				//Sanity check av state-endring
				tmp := states[msg.Sender]
				if msg.Type == com.StoppingToFinishOrder {
					driver.SetBtnLamp(msg.NewState.LastPassedFloor, driver.Up, false)
					driver.SetBtnLamp(msg.NewState.LastPassedFloor, driver.Down, false)
				}
				statetype.DeepOrdersetCopy(msg.NewState.Orders, tmp.Orders)
				tmp.Direction = msg.NewState.Direction
				tmp.LastPassedFloor = msg.NewState.LastPassedFloor
				tmp.Moving = msg.NewState.Moving
				tmp.SequenceNumber = msg.NewState.SequenceNumber
				tmp.DoorOpen = msg.NewState.DoorOpen
				states[msg.Sender] = tmp
			case com.InitialStateMsg:
				//Sanity check av state-endring
				if _, ok := states[msg.Sender]; !ok {
					fmt.Println("Manager error: Received InitialStateMsg without first getting message on connected_chan")
					break
				}
				if states[msg.Sender].Valid {
					break
				}

				fmt.Println("\033[34m"+"Manager: received InitialStateMsg from", msg.Sender, "\033[0m")
				tmp := states[msg.Sender]
				tmp.LastPassedFloor = msg.NewState.LastPassedFloor
				tmp.Direction = msg.NewState.Direction
				tmp.Moving = msg.NewState.Moving
				tmp.SequenceNumber = msg.NewState.SequenceNumber
				tmp.DoorOpen = msg.NewState.DoorOpen
				tmp.Valid = true
				states[msg.Sender] = tmp
				statetype.DeepOrdersetCopy(msg.NewState.Orders, states[msg.Sender].Orders)

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
			fmt.Println("\033[34m"+"Manager: Order(s) at floor", completed, "finished"+"\033[0m")
			tmp := states[localIp]
			tmp.Moving = false
			tmp.DoorOpen = true
			tmp.Orders.ClearOrders(completed)
			driver.SetBtnLamp(completed, driver.Up, false)
			driver.SetBtnLamp(completed, driver.Down, false)
			driver.SetBtnLamp(completed, driver.Command, false)
			states[localIp] = tmp
			send_chan <- com.SensorEventMsg{com.StoppingToFinishOrder, states[localIp].CreateCopy(), localIp}
		case <-door_closed_chan:
			tmp := states[localIp]
			tmp.DoorOpen = false
			states[localIp] = tmp
			send_chan <- com.SensorEventMsg{com.DoorClosed, states[localIp].CreateCopy(), localIp}
		case connected := <-connected_chan:
			states[connected] = statetype.State{-1, elevator.Up, false, make(statetype.Orderset), false, 0, false}
			states[connected].Orders[driver.Up] = make(statetype.FloorOrders)
			states[connected].Orders[driver.Down] = make(statetype.FloorOrders)
			states[connected].Orders[driver.Command] = make(statetype.FloorOrders)
			fmt.Println("\033[34m"+"Manager: sending InitialStateMsg to", connected, "\033[0m")
			send_chan <- com.InitialStateMsg{states[localIp].CreateCopy(), localIp}

		case disconnected := <-disconnected_chan:

			for order := range unconfirmedOrders {
				if order.Reciever == disconnected {
					go func(btnClick driver.ClickEvent) {
						clickEvent_chan <- btnClick
					}(order.Button)
				}
			}

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
					if btnType != driver.Command {
						for floor, isSet := range floorOrders {
							if isSet {
								// Find the elevator with shortest expected response time
								bestIp := localIp                          // Local elevator is default
								var shortestResponseTime float32 = 99999.9 //Inf
								for ip, state := range states {
									if time := state.GetExpectedResponseTime(driver.ClickEvent{floor, btnType}); time < shortestResponseTime {
										shortestResponseTime = time
										bestIp = ip
									}
								}
								// Send order to the best elevator
								fmt.Println("\033[34m"+"Manager: Reassigning order floor", floor, ",type", btnType, "to ip", bestIp, "\033[0m")
								if bestIp == localIp {
									tmp := states[localIp]
									tmp.SequenceNumber += 1
									tmp.Orders.AddOrder(driver.ClickEvent{floor, btnType})
									states[localIp] = tmp
									send_chan <- com.OrderEventMsg{driver.ClickEvent{floor, btnType}, states[localIp].CreateCopy(), localIp}
								} else {
									send_chan <- com.OrderAssignmentMsg{driver.ClickEvent{floor, btnType}, bestIp, localIp}
									unconfirmedOrders[unconfirmedOrder{driver.ClickEvent{floor, btnType}, bestIp}] = true
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
			if buttonClick.Type == driver.Command {
				if !states[localIp].Orders.IsOrder(buttonClick) {
					states[localIp].Orders.AddOrder(buttonClick)
					driver.SetBtnLamp(buttonClick.Floor, buttonClick.Type, true)
					//statetype.SaveInternalOrder(buttonClick.Floor)
					//Add to local state
					tmp := states[localIp]
					tmp.SequenceNumber += 1
					states[localIp] = tmp
					send_chan <- com.OrderEventMsg{buttonClick, states[localIp].CreateCopy(), localIp}
					fmt.Println("\033[34m"+"Manager: New internal order at floor", buttonClick.Floor, "\033[0m")
				}
			} else {
				exists := false
				for _, state := range states {
					if state.Orders.IsOrder(buttonClick) {
						fmt.Println("\033[34m"+"Order already exists:", buttonClick, "\033[0m")
						exists = true
						break
					}
				}
				if exists {
					break
				}
				fmt.Println("\033[34m"+"Manager: New", buttonClick, "\033[0m")
				bestIp := localIp                          // Local elevator is default
				var shortestResponseTime float32 = 99999.9 //Inf
				for ip, state := range states {
					fmt.Println("IP:", ip)
					if time := state.GetExpectedResponseTime(buttonClick); time < shortestResponseTime {
						shortestResponseTime = time
						bestIp = ip
					}
				}
				fmt.Println("Best IP for", buttonClick, "is", bestIp)
				if bestIp == localIp {
					tmp := states[localIp]
					tmp.SequenceNumber += 1
					tmp.Orders.AddOrder(buttonClick)
					states[localIp] = tmp
					send_chan <- com.OrderEventMsg{buttonClick, states[localIp].CreateCopy(), localIp}
					driver.SetBtnLamp(buttonClick.Floor, buttonClick.Type, true)
				} else {
					send_chan <- com.OrderAssignmentMsg{buttonClick, bestIp, localIp}
					unconfirmedOrders[unconfirmedOrder{buttonClick, bestIp}] = true
				}

			}
		case sensorEvent := <-sensorEvent_chan:
			fmt.Println("\033[34m"+"Sensorevent:", sensorEvent, "\033[0m")
			if sensorEvent == -1 && !states[localIp].Moving {
				elev_error_chan <- true
				continue
			}
			if sensorEvent == -1 {
				tmp := states[localIp]
				tmp.SequenceNumber += 1
				states[localIp] = tmp
				send_chan <- com.SensorEventMsg{com.LeavingFloor, states[localIp].CreateCopy(), localIp}
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
			fmt.Println("\033[34m" + "Manager: Starting to move" + "\033[0m")
			tmp := states[localIp]
			tmp.Moving = true
			states[localIp] = tmp
		case <-PassingFloor:
			fmt.Println("\033[34m" + "Manager: Passing floor" + "\033[0m")
			send_chan <- com.SensorEventMsg{com.PassingFloor, states[localIp].CreateCopy(), localIp}
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
