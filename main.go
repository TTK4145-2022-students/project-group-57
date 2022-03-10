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

func main() {
	SlaveButtonRx := make(chan SlaveButtonEventMsg)
	SlaveFloorRx := make(chan int)
	MasterCommandMD := make(chan int)

	go broadcast.Receiver(16513, SlaveButtonRx)
	go broadcast.Receiver(16514, SlaveFloorRx)
	go broadcast.Transmitter(16515, MasterCommandMD)

	e1 := elevator.Elevator{
		Floor:     -1,
		Dirn:      -1,
		Requests:  [elevio.NumFloors][elevio.NumButtonTypes]bool{},
		Behaviour: -1,
	}

	//Testing
	e1.Requests[3][2] = true
	//Testing

	for {
		select {
		case slaveMsg := <-SlaveButtonRx:
			fmt.Println("Floor")
			fmt.Println(slaveMsg.Btn_floor)
			fmt.Println("Button type")
			fmt.Println(slaveMsg.Btn_type)
			fmt.Println(" ")
			e1.Requests[slaveMsg.Btn_floor][slaveMsg.Btn_type] = true

		case slaveMsg := <-SlaveFloorRx:
			fmt.Println("Arrived at floor:")
			fmt.Println(int(slaveMsg))
			fmt.Println(" ")
			e1.Floor = slaveMsg
			if requests.RequestsHere(e1) {
				MasterMotorDirTx := elevio.MD_Stop
				MasterCommandMD <- MasterMotorDirTx
				fmt.Println("Elevator should stop!")
			}
		}

	}

}
