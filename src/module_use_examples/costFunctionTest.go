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

	From testing at the lab:
	Travel between floors: 2.2
	Passing floor: 0.385

*/

func main(){
	//Create test state
	upOrders := statetype.FloorOrders{0:false , 1: false, 2: false}
	downOrders := statetype.FloorOrders{1: false, 2: true, 3: false}
	CmdOrders := statetype.FloorOrders{0: false, 1: false, 2: false, 3: false}
	orders := statetype.Orderset{
		simdriver.Up: upOrders,
		simdriver.Down: downOrders,
		simdriver.Command: CmdOrders}
	testState := statetype.State{
		3, // Last passed floor
		elevator.Down, // Defined direction
		true, //moving
		orders,
		true, //valid
		0, //Sequence number
		false} // Door open

	//Create new test order
	newOrder := simdriver.ClickEvent{3,simdriver.Down}

	//Calculate times
	response, best, worst := testState.GetExpectedResponseTime(newOrder)
	fmt.Println("Response time:",response)
	fmt.Println("Best case:",best)
	fmt.Println("Worst case:",worst)
	
}