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

func main() {
	slaveButtonRx := make(chan SlaveButtonEventMsg)
	slaveFloorRx := make(chan int)
	masterCommandMD := make(chan int)
	masterAckOrder := make(chan MasterAckOrderMsg)
	slaveAckOrderDoneRx := make(chan bool)
	masterTurnOffOrderLightTx := make(chan int)

	go broadcast.Receiver(16513, slaveButtonRx)
	go broadcast.Receiver(16514, slaveFloorRx)
	go broadcast.Transmitter(16515, masterCommandMD)
	go broadcast.Transmitter(16516, masterAckOrder)
	go broadcast.Receiver(16517, slaveAckOrderDoneRx)
	go broadcast.Transmitter(16518, masterTurnOffOrderLightTx)

	e1 := elevator.Elevator{
		Floor:     -1,
		Dirn:      -1,
		Requests:  [elevio.NumFloors][elevio.NumButtonTypes]bool{},
		Behaviour: -1,
	}

	//Testing
	//e1.Requests[3][2] = true
	//Testing

	for {
		select {
		case slaveMsg := <-slaveButtonRx:
			fmt.Println("Floor")
			fmt.Println(slaveMsg.Btn_floor)
			fmt.Println("Button type")
			fmt.Println(slaveMsg.Btn_type)
			fmt.Println(" ")
			e1.Requests[slaveMsg.Btn_floor][slaveMsg.Btn_type] = true
			masterAckOrder <- MasterAckOrderMsg{int(slaveMsg.Btn_floor), slaveMsg.Btn_type}
			if requests.RequestsHere(e1) {
				masterCommandMD <- elevio.MD_Stop
				fmt.Println("Elevator should stop!")
			} else if requests.RequestsAbove(e1) {
				masterCommandMD <- int(elevio.MD_Up)
			} else if requests.RequestsBelow(e1) {
				masterCommandMD <- int(elevio.MD_Down)
			}

		case slaveMsg := <-slaveFloorRx:
			fmt.Println("Arrived at floor:")
			fmt.Println(int(slaveMsg))
			fmt.Println(" ")
			e1.Floor = slaveMsg
			if requests.RequestsHere(e1) {
				masterCommandMD <- elevio.MD_Stop
				fmt.Println("Elevator should stop!")
			} else if requests.RequestsAbove(e1) {
				masterCommandMD <- int(elevio.MD_Up)
			} else if requests.RequestsBelow(e1) {
				masterCommandMD <- int(elevio.MD_Down)
			}

		case slaveMsg := <-slaveAckOrderDoneRx:
			if slaveMsg {
				e1.Requests[e1.Floor][0] = false
				e1.Requests[e1.Floor][1] = false
				e1.Requests[e1.Floor][2] = false

				masterTurnOffOrderLightTx <- e1.Floor
			}

		}
	}

}
