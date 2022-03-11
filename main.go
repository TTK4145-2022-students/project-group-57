package main

//TODO
//Dumb down slave
//Maybe remove states and use channels????

//Make master decide and set direction for slave (testing)
//Send requests from slave to master
//Save requests in a reasonable format in master - with states?
//Update slave-state and save in a reasonable format in master

//Slave executes its own cab requests

//Fix logic in request-execution in master
//Master assigns request, divides it in to the simplest steps, saves which elevator executes the request.

//Message from slave to master -
//Need to include: New buttonevents + new floor + iterator/versionnr
//Alive message

//Message from master to slave -
//Need to include: New motor direction + open door + set lights

//Master-message
//Need to include: All relevant info + iterate/versionnr

import (
	"fmt"
	"master/Driver-go/elevio"
	"master/elevator"
	"master/network/broadcast"
	"master/requests"
)

type SlaveButtonEventMsg struct {
	Btn_floor int
	Btn_type  int
}

type MasterAckOrderMsg struct {
	Btn_floor int
	Btn_type  int
}

var e1 elevator.Elevator
var MasterRequests requests.AllRequests

func main() {

	/*//Maybe init func?
	e1 := elevator.Elevator{
		Floor:     elevio.GetFloor(),
		Dirn:      elevio.MD_Stop,
		Requests:  [elevio.NumFloors][elevio.NumButtonTypes]bool{},
		Behaviour: elevator.EB_Idle,
	}

	if e1.Floor == -1 {
		e1 = fsm.Fsm_onInitBetweenFloors(e1)
	}

	fsm.SetAllLights(e1)
	var initFloor int
	broadcast.Receiver(16519, initFloor)
	fmt.Println(initFloor)*/

	slaveButtonRx := make(chan SlaveButtonEventMsg)
	slaveFloorRx := make(chan int)
	masterCommandMD := make(chan int)
	masterAckOrder := make(chan MasterAckOrderMsg)
	slaveAckOrderDoneRx := make(chan bool)
	masterTurnOffOrderLightTx := make(chan int)
	slaveState := make(chan elevator.Elevator)

	go broadcast.Receiver(16513, slaveButtonRx)
	go broadcast.Receiver(16514, slaveFloorRx)
	go broadcast.Transmitter(16515, masterCommandMD)
	go broadcast.Transmitter(16516, masterAckOrder)
	go broadcast.Receiver(16517, slaveAckOrderDoneRx)
	go broadcast.Transmitter(16518, masterTurnOffOrderLightTx)
	go broadcast.Receiver(16519, slaveState)

	//Testing
	//e1.Requests[3][2] = true
	//Testing

	for {
		/* fmt.Println("Floor from state: ")
		fmt.Println(e1.Floor)
		fmt.Println("Behaviour: ")
		fmt.Println(e1.Behaviour)*/

		/*if e1.Behaviour == elevator.EB_Idle {
			a := requests.MasterRequestsNextAction(e1, MasterRequests)
			e1.Dirn = a.Dirn
			e1.Behaviour = a.Behaviour

			switch e1.Behaviour {
			case elevator.EB_DoorOpen:
				e1 = requests.ClearRequestCurrentFloor(e1)
				fsm.SetAllLights(e1)
			case elevator.EB_Moving:
				masterCommandMD <- int(e1.Dirn)
			case elevator.EB_Idle:
				masterCommandMD <- int(e1.Dirn)
			}
		}*/

		select {
		case localState := <-slaveState:
			e1 = localState
		case slaveMsg := <-slaveButtonRx: //New order from slave
			fmt.Println("Floor")
			fmt.Println(slaveMsg.Btn_floor)
			fmt.Println("Button type")
			fmt.Println(slaveMsg.Btn_type)
			fmt.Println(" ")

			MasterRequests.Requests[slaveMsg.Btn_floor][slaveMsg.Btn_type] = true //Saving requests in master
			e1.Requests[slaveMsg.Btn_floor][slaveMsg.Btn_type] = true
			fmt.Println(MasterRequests)

			masterAckOrder <- MasterAckOrderMsg{int(slaveMsg.Btn_floor), slaveMsg.Btn_type}

			if requests.MasterRequestsHere(e1, MasterRequests) {
				masterCommandMD <- elevio.MD_Stop
				fmt.Println("Elevator should stop!")
			} else if requests.MasterRequestsAbove(e1, MasterRequests) {
				masterCommandMD <- int(elevio.MD_Up)
			} else if requests.MasterRequestsBelow(e1, MasterRequests) {
				masterCommandMD <- int(elevio.MD_Down)
			}
		case slaveMsg := <-slaveFloorRx: //New floor from slave
			fmt.Println("Arrived at floor:")
			fmt.Println(int(slaveMsg))
			fmt.Println(" ")
			e1.Floor = slaveMsg

			if requests.MasterRequestShouldStop(e1, MasterRequests) {
				masterCommandMD <- int(elevio.MD_Stop)
			}

			a := requests.MasterRequestsNextAction(e1, MasterRequests)
			e1.Dirn = a.Dirn
			e1.Behaviour = a.Behaviour

			switch e1.Behaviour {
			case elevator.EB_DoorOpen:
				e1, MasterRequests = requests.MasterClearRequestCurrentFloor(e1, MasterRequests)
				//fsm.SetAllLights(e1)
				//elevio.SetButtonLamp(elevio.BT_HallDown, e.Floor, false)
				masterTurnOffOrderLightTx <- e1.Floor
			case elevator.EB_Moving:
				masterCommandMD <- int(e1.Dirn)
			case elevator.EB_Idle:
				masterCommandMD <- int(e1.Dirn)
			}

			if requests.RequestsHere(e1) {
				masterCommandMD <- elevio.MD_Stop
				fmt.Println("Elevator should stop!")
			} else if requests.RequestsAbove(e1) {
				masterCommandMD <- int(elevio.MD_Up)
			} else if requests.RequestsBelow(e1) {
				masterCommandMD <- int(elevio.MD_Down)
			}

		case slaveMsg := <-slaveAckOrderDoneRx: //Recieve ack from slave, order done
			if slaveMsg {
				MasterRequests.Requests[e1.Floor][0] = false
				MasterRequests.Requests[e1.Floor][1] = false
				MasterRequests.Requests[e1.Floor][2] = false
				fmt.Println(MasterRequests)

				e1.Requests[e1.Floor][0] = false
				e1.Requests[e1.Floor][1] = false
				e1.Requests[e1.Floor][2] = false

				masterTurnOffOrderLightTx <- e1.Floor
			}

		}
	}

}

//Add check to see if order queue is empty or if new order need to be executed

//Dont tell elevator to move inside buttonevent

//include functionality from onDoorTimeout to check if elevator should continue
//moving after opening door in a floor
