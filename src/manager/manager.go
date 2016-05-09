package manager

import (
	"../com"
	"../driver"
	"../elevator"
	"../networking"
	"../state"
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
	clickEvent chan driver.ClickEvent,
	sensorEvent <-chan int,
	stopBtnEvent <-chan bool,
	completedFloor <-chan int,
	floorReached chan<- int,
	newDirection <-chan elevator.Direction,
	doorClosed <-chan bool,
	startedMoving <-chan bool,
	passingFloor <-chan bool,
	elevatorError <-chan bool,
	resumeAfterError chan<- bool,
	externalError chan<- bool,
	readDirection <-chan elevator.ReadDirection,
	readOrder <-chan elevator.ReadOrder,
	sendMsg chan<- com.Message,
	recvMsg <-chan com.Message,
	connected <-chan string,
	disconnected <-chan string,
	setNetworkStatus chan<- bool) {

	localIp = networking.GetLocalIp()
	errorState := false

	unconfirmedOrders := make(map[unconfirmedOrder]bool)
	redistributedOrders := make(map[driver.ClickEvent]bool)

	states := make(map[string]state.State)
	initializeState(states, localIp)

	states[localIp].Orders.Init()
	states[localIp].Orders.RestoreInternalOrders()

	for {
		select {
		case msg := <-recvMsg:
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
				sendMsg <- com.OrderEventMsg{msg.Button, states[localIp].CreateCopy(), localIp}

			case com.OrderEventMsg:
				delete(unconfirmedOrders, unconfirmedOrder{msg.Button, msg.Sender})
				delete(redistributedOrders, msg.Button)
				states[msg.Sender].Orders.AddOrder(msg.Button)
				if msg.Button.Type != driver.Command {
					driver.SetBtnLamp(msg.Button.Floor, msg.Button.Type, true)
				}
			case com.SensorEventMsg:
				tmpState := states[msg.Sender]
				if msg.Type == com.StoppingToFinishOrder {
					driver.SetBtnLamp(msg.NewState.LastPassedFloor, driver.Up, false)
					driver.SetBtnLamp(msg.NewState.LastPassedFloor, driver.Down, false)
				}
				state.DeepOrdersetCopy(msg.NewState.Orders, tmpState.Orders)
				tmpState.Direction = msg.NewState.Direction
				tmpState.LastPassedFloor = msg.NewState.LastPassedFloor
				tmpState.Moving = msg.NewState.Moving
				tmpState.DoorOpen = msg.NewState.DoorOpen
				states[msg.Sender] = tmpState
			case com.InitialStateMsg:
				if _, ok := states[msg.Sender]; !ok {
					fmt.Println("Manager error: Received InitialStateMsg without first getting message on connected")
					break
				}
				if states[msg.Sender].Valid {
					break
				}

				fmt.Println("\033[34m"+"Manager: received InitialStateMsg from", msg.Sender, "\033[0m")
				tmpState := states[msg.Sender]
				tmpState.LastPassedFloor = msg.NewState.LastPassedFloor
				tmpState.Direction = msg.NewState.Direction
				tmpState.Moving = msg.NewState.Moving
				tmpState.DoorOpen = msg.NewState.DoorOpen
				tmpState.Valid = true
				states[msg.Sender] = tmpState
				state.DeepOrdersetCopy(msg.NewState.Orders, states[msg.Sender].Orders)
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

		case readOrder := <-readOrder:
			readOrder.Resp <- states[localIp].Orders.IsOrder(readOrder.Order)
		case readDir := <-readDirection:
			switch readDir.Request {
			case elevator.IsOrderAhead:
				readDir.Resp <- states[localIp].Orders.IsOrderAhead(readDir.Floor, readDir.Dir)
			case elevator.IsOrderBehind:
				readDir.Resp <- states[localIp].Orders.IsOrderBehind(readDir.Floor, readDir.Dir)
			}

		case completed := <-completedFloor:
			fmt.Println("\033[34m"+"Manager: Order(s) at floor", completed, "finished"+"\033[0m")
			tmpState := states[localIp]
			tmpState.Moving = false
			tmpState.DoorOpen = true
			tmpState.Orders.ClearOrders(completed)
			driver.SetBtnLamp(completed, driver.Up, false)
			driver.SetBtnLamp(completed, driver.Down, false)
			driver.SetBtnLamp(completed, driver.Command, false)
			states[localIp] = tmpState
			state.DeleteSavedOrder(completed)
			sendMsg <- com.SensorEventMsg{com.StoppingToFinishOrder, states[localIp].CreateCopy(), localIp}

		case <-doorClosed:
			tmpState := states[localIp]
			tmpState.DoorOpen = false
			states[localIp] = tmpState
			sendMsg <- com.SensorEventMsg{com.DoorClosed, states[localIp].CreateCopy(), localIp}

		case connected := <-connected:
			initializeState(states, connected)
			fmt.Println("\033[34m"+"Manager: sending InitialStateMsg to", connected, "\033[0m")
			sendMsg <- com.InitialStateMsg{states[localIp].CreateCopy(), localIp}

		case disconnected := <-disconnected:
			fmt.Println("\033[34m"+"Disconnected:", disconnected, "\033[0m")

			highestIp := getHighestIp(states)
			secondHighestIp, ok := getSecondHighestIp(states)
			shouldRedistribute := (highestIp == localIp || (highestIp == disconnected && localIp == secondHighestIp))

			//Redistribute unconfirmed orders
			for order := range unconfirmedOrders {
				if order.Receiver == disconnected {
					go func(btnClick driver.ClickEvent) {
						clickEvent <- btnClick
					}(order.Button)
				}
			}

			//redistribute orders where the responsible redistributor has died
			fmt.Println("\033[34m"+"\tManager: highest ip:", highestIp, "secondHighestIp:", secondHighestIp, "\033[0m")
			if disconnected == highestIp && localIp == secondHighestIp && ok {
				fmt.Println("\033[34m" + "Manager: highest ip disconnected, redistributing redistributed orders" + "\033[0m")
				for button := range redistributedOrders {
					delete(redistributedOrders, button)
					go func(btnClick driver.ClickEvent) {
						clickEvent <- btnClick
					}(button)
				}
			}

			// Redistribute normal orders, or add them to the list of orders being redistributed by someone else
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
									clickEvent <- btnClick
								}(driver.ClickEvent{floor, btnType})
							} else {
								redistributedOrders[driver.ClickEvent{floor, btnType}] = true
							}
						}
					}
				}
			}

			delete(states, disconnected)

		case <-stopBtnEvent:
			if errorState {
				resumeAfterError <- true
				setNetworkStatus <- true
				errorState = false
				driver.SetStopLamp(false)
			}

		case buttonClick := <-clickEvent:
			if errorState {
				break
			}
			if buttonClick.Type == driver.Command {
				if !states[localIp].Orders.IsOrder(buttonClick) {
					states[localIp].Orders.AddOrder(buttonClick)
					driver.SetBtnLamp(buttonClick.Floor, buttonClick.Type, true)
					state.SaveInternalOrder(buttonClick.Floor)
					sendMsg <- com.OrderEventMsg{buttonClick, states[localIp].CreateCopy(), localIp}
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
					tmpState := states[localIp]
					tmpState.Orders.AddOrder(buttonClick)
					states[localIp] = tmpState
					sendMsg <- com.OrderEventMsg{buttonClick, states[localIp].CreateCopy(), localIp}
					driver.SetBtnLamp(buttonClick.Floor, buttonClick.Type, true)
				} else {
					sendMsg <- com.OrderAssignmentMsg{buttonClick, bestIp, localIp}
					unconfirmedOrders[unconfirmedOrder{buttonClick, bestIp}] = true
				}
			}

		case sensorEvent := <-sensorEvent:
			if errorState {
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
				sendMsg <- com.SensorEventMsg{com.LeavingFloor, states[localIp].CreateCopy(), localIp}
			} else {
				tmpState := states[localIp]
				tmpState.LastPassedFloor = sensorEvent
				if !tmpState.Valid {
					tmpState.Valid = true
				} else {
					if states[localIp].Moving {
						floorReached <- sensorEvent
					}
				}
				states[localIp] = tmpState
			}

		case <-startedMoving:
			fmt.Println("\033[34m" + "Manager: Starting to move" + "\033[0m")
			tmpState := states[localIp]
			tmpState.Moving = true
			states[localIp] = tmpState

		case direction := <-newDirection:
			tmpState := states[localIp]
			tmpState.Direction = direction
			states[localIp] = tmpState

		case <-passingFloor:
			fmt.Println("\033[34m" + "Manager: Passing floor" + "\033[0m")
			sendMsg <- com.SensorEventMsg{com.PassingFloor, states[localIp].CreateCopy(), localIp}

		case <-elevatorError:
			fmt.Println("\033[34m" + "Manager: Entering error state" + "\033[0m")
			errorState = true
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

func initializeState(states map[string]state.State, ip string) {
	states[ip] = state.State{-1, elevator.GetCurrentDirection(), false, make(state.Orderset), false, false}
	states[ip].Orders[driver.Up] = make(state.FloorOrders)
	states[ip].Orders[driver.Down] = make(state.FloorOrders)
	states[ip].Orders[driver.Command] = make(state.FloorOrders)
}

func getHighestIp(states map[string]state.State) string {
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

func getSecondHighestIp(states map[string]state.State) (string, bool) {
	ips := make([]int, 0)
	ipMap := make(map[int]string)
	for ip, _ := range states {
		ipParts := strings.SplitAfter(ip, ".")

		if len(ipParts) != 4 {
			return "", false
		}

		ipInt, err := strconv.Atoi(ipParts[3])

		if err != nil {
			return "", false
		}

		ipMap[ipInt] = ip
		ips = append(ips, ipInt)
	}
	if len(ips) < 2 {
		return "", false
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ips)))
	return ipMap[ips[1]], true
}
