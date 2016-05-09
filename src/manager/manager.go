package manager

import (
	"../com"
	driver "../driver"
	"../elevator"
	"../networking"
	"../statetype"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type unconfirmedOrder struct {
	Button   driver.ClickEvent
	Receiver string
}

var localIp string

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
	new_direction_chan <-chan elevator.Direction_t,
	PassingFloor <-chan bool,
	elev_error_chan chan bool,
	setNetworkStatus chan<- bool,
	resumeAfterError chan<- bool,
	stopButtonChan <-chan bool,
	externalError chan<- bool) {

	localIp = networking.GetLocalIp()
	error_state := false

	unconfirmedOrders := make(map[unconfirmedOrder]bool)
	redistributedOrders := make(map[driver.ClickEvent]bool)

	//Initialize queue
	states := make(map[string]statetype.State)
	states[localIp] = statetype.State{-1, elevator.GetCurrentDirection(), false, make(statetype.Orderset), false, 0, false}
	states[localIp].Orders[driver.Up] = make(statetype.FloorOrders)
	states[localIp].Orders[driver.Down] = make(statetype.FloorOrders)
	states[localIp].Orders[driver.Command] = make(statetype.FloorOrders)
	states[localIp].Orders.Init()
	states[localIp].Orders.RestoreInternalOrders()

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
				delete(redistributedOrders, msg.Button)
				driver.SetBtnLamp(msg.Button.Floor, msg.Button.Type, true)
				send_chan <- com.OrderEventMsg{msg.Button, states[localIp].CreateCopy(), localIp}

			case com.OrderEventMsg:
				if msg.NewState.SequenceNumber <= states[msg.Sender].SequenceNumber {
					fmt.Println("Received OrderEventMsg with nonincreasing sequence number")
					break
				}
				delete(unconfirmedOrders, unconfirmedOrder{msg.Button, msg.Sender})
				delete(redistributedOrders, msg.Button)
				states[msg.Sender].Orders.AddOrder(msg.Button)
				if msg.Button.Type != driver.Command {
					driver.SetBtnLamp(msg.Button.Floor, msg.Button.Type, true)
				}
			case com.SensorEventMsg:
				if msg.NewState.SequenceNumber <= states[msg.Sender].SequenceNumber {
					fmt.Println("Received SensorEventMsg with nonincreasing sequence number")
					break
				}
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
				for btnType, floorOrders := range states[msg.Sender].Orders {
					if btnType != driver.Command {
						for floor, isSet := range floorOrders {
							if isSet {
								driver.SetBtnLamp(floor, btnType, true)
							}
						}
					}
				}

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
			statetype.DeleteSavedOrder(completed)
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
			fmt.Println("\033[34m"+"Disconnected:", disconnected, "\033[0m")
			//Redistribute unconfirmed orders
			for order := range unconfirmedOrders {
				if order.Receiver == disconnected {
					go func(btnClick driver.ClickEvent) {
						clickEvent_chan <- btnClick
					}(order.Button)
				}
			}

			highestIp := getHighestIp(states)
			secondHighestIp, ok := getSecondHighestIp(states)
			fmt.Println("\033[34m"+"\tManager: highest ip:", highestIp, "secondHighestIp:", secondHighestIp, "\033[0m")
			if disconnected == highestIp && localIp == secondHighestIp && ok {
				//redistribute redistributed orders
				fmt.Println("\033[34m" + "Manager: highest ip disconnected, redistributing redistributed orders" + "\033[0m")
				for button := range redistributedOrders {
					delete(redistributedOrders, button)
					go func(btnClick driver.ClickEvent) {
						clickEvent_chan <- btnClick
					}(button)
				}
			}

			// For every external order that needs to be redistributed
			shouldRedistribute := (highestIp == localIp || (highestIp == disconnected && localIp == secondHighestIp))
			if shouldRedistribute {
				fmt.Println("\033[34m" + "\tRedistributing" + "\033[0m")
			} else {
				fmt.Println("\033[34m" + "\tWe should not redistribute, adding orders to redistributed orders" + "\033[0m")
			}

			for btnType, floorOrders := range states[disconnected].Orders {
				if btnType != driver.Command {
					for floor, isSet := range floorOrders {
						if isSet {
							if shouldRedistribute {
								go func(btnClick driver.ClickEvent) {
									clickEvent_chan <- btnClick
								}(driver.ClickEvent{floor, btnType})
							} else {
								redistributedOrders[driver.ClickEvent{floor, btnType}] = true
							}
						}
					}
				}
			}
			delete(states, disconnected)

		case <-stopButtonChan:
			if error_state {
				resumeAfterError <- true
				setNetworkStatus <- true
				error_state = false
				driver.SetStopLamp(false)
			}

		case buttonClick := <-clickEvent_chan:
			if error_state {
				break
			}
			if buttonClick.Type == driver.Command {
				if !states[localIp].Orders.IsOrder(buttonClick) {
					states[localIp].Orders.AddOrder(buttonClick)
					driver.SetBtnLamp(buttonClick.Floor, buttonClick.Type, true)
					statetype.SaveInternalOrder(buttonClick.Floor)
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
			if error_state {
				break
			}
			fmt.Println("\033[34m"+"Sensorevent", sensorEvent, "\033[0m")
			if sensorEvent == -1 && !states[localIp].Moving {
				fmt.Println("\033[34m" + "Manager: left floor without moving" + "\033[0m")
				go func() {
					externalError <- true
				}()
				break
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
					if states[localIp].Moving {
						floor_reached <- sensorEvent
					}
				}
				states[localIp] = tmp
			}

		case <-start_moving:
			fmt.Println("\033[34m" + "Manager: Starting to move" + "\033[0m")
			tmp := states[localIp]
			tmp.Moving = true
			states[localIp] = tmp

		case direction := <-new_direction_chan:
			tmp := states[localIp]
			tmp.Direction = direction
			states[localIp] = tmp

		case <-PassingFloor:
			fmt.Println("\033[34m" + "Manager: Passing floor" + "\033[0m")
			send_chan <- com.SensorEventMsg{com.PassingFloor, states[localIp].CreateCopy(), localIp}

		case <-elev_error_chan:
			fmt.Println("\033[34m" + "Manager: Entering error state" + "\033[0m")
			error_state = true
			setNetworkStatus <- false
			for ip, _ := range states {
				if ip != localIp {
					delete(states, ip)
				}
			}
			driver.SetStopLamp(true)
		}
	}
}

func getHighestIp(states map[string]statetype.State) string {
	highestIp := localIp
	for ip, _ := range states {
		remoteIpHighest, err := networking.HasHighestIp(ip, highestIp)
		if err != nil {
			fmt.Println(err)
		}
		if remoteIpHighest {
			highestIp = ip
		}
	}
	return highestIp
}

func getSecondHighestIp(states map[string]statetype.State) (string, bool) {
	ips := make([]int, 0)
	ipMap := make(map[int]string)
	for ip, _ := range states {
		ipParts := strings.SplitAfter(ip, ".")

		if len(ipParts) != 4 {
			return "", false
		}

		ip_int, err := strconv.Atoi(ipParts[3])

		if err != nil {
			return "", false
		}

		ipMap[ip_int] = ip
		ips = append(ips, ip_int)
	}
	if len(ips) < 2 {
		return "", false
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ips)))
	return ipMap[ips[1]], true
}
