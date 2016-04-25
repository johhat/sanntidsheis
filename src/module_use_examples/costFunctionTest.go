package main

import(
	"fmt"
	"../statetype"
	"../elevator"
	"../simdriver"
)
/*
	stopTime float32 = 3
	floorTravelTime float32 = 2
	movingPenalty float32 = floorTravelTime/2
	doorOpenPenalty float32 = stopTime/2
*/

func main(){
	//Create test state
	upOrders := statetype.FloorOrders{0: false, 1: false, 2: false}
	downOrders := statetype.FloorOrders{1: false, 2: false, 3: false}
	CmdOrders := statetype.FloorOrders{0: false, 1: false, 2: false, 3: false}
	orders := statetype.Orderset{
		simdriver.Up: upOrders,
		simdriver.Down: downOrders,
		simdriver.Command: CmdOrders}
	testState := statetype.State{
		0,
		elevator.Up,
		false, //moving
		orders,
		true,
		0,
		false}

	//Create new test order
	newOrder := simdriver.ClickEvent{3,simdriver.Down}

	//Calculate times
	response, best, worst := testState.GetExpectedResponseTime(newOrder)
	fmt.Println("Response time:",response)
	fmt.Println("Best case:",best)
	fmt.Println("Worst case:",worst)
	
}