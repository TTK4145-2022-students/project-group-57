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
	"master/network/broadcast"
)

type SlaveButtonEventMsg struct {
	Btn_floor int
	Btn_type  int
	//Iter    int
}

type HelloMsg struct {
	Message string
	Iter    int
}

func main() {
	SlaveRx := make(chan SlaveButtonEventMsg)
	helloTx := make(chan HelloMsg)

	go broadcast.Receiver(16513, SlaveRx)
	go broadcast.Transmitter(16514, helloTx)

	for {

		slaveMsg := <-SlaveRx

		goodbyeMsg := HelloMsg{"I am master", 0}
		helloTx <- goodbyeMsg

		fmt.Println(slaveMsg.Btn_floor)
		fmt.Println(slaveMsg.Btn_type)

	}

}
