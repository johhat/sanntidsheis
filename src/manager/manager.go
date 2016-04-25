package manager

import (
	"fmt"
	"../simdriver"
	"../elevator"
	"../networking"
	"../statetype"
	"../com"
	"os"
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
					}
				case com.ClickEventMsg:
					//Sanity check av state-endring
				case com.SensorEventMsg:
					//Sanity check av state-endring
				case com.InitialStateMsg:
					//Sanity check av state-endring
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
				for ip, _ := range states {
					if ip > localIp && ip != disconnected{
						//Fjern heisen fra states
						break
					}
				}
				//Vi refordeler alle eksterne ordre
					//Sjekk forventet responstid for alle heiser, velg den beste for hver ordre
				//Fjern heisen fra states
			case ip := <-newElevator:
				states[ip] = statetype.State{-1, elevator.Up, false, make(statetype.Orderset), false}
				states[ip].orders[simdriver.Up] = make(statetype.FloorOrders)
				states[ip].orders[simdriver.Down] = make(statetype.FloorOrders)
				states[ip].orders[simdriver.Command] = make(statetype.FloorOrders)
				//send_chan<- //send state
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
					//send_chan <- //Varsle om ny state
				} else {
					//Sjekk forventet responstid for alle heiser, velg den beste
				}
			case sensorEvent := <-sensorEvent_chan:
				//
			case <-elev_error_chan:
				// disconnect TCP
		}
	}
}

func saveInternalOrder(floor int){
	f, err := os.Create("/internalOrder"+floor)
}





func sanityCheck(oldState statetype.State, newState statetype.State, event Event) bool {
	//Check if old state + event = new state
	switch event {
	case com.NewExternalOrder:
		//
	case com.NewInternalOrder:
		//
	case com.SelfAssignedOrder:
		//
	case com.PassingFloor:
		//
	case com.DoorOpenedByInternalOrder:
		//
	case com.StoppingToFinishOrder:
		//
	case com.LeavingFloor:
		//
	}
	return true
}
