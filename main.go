package main

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

//Logic need to use MasterRequest and not ElevatorRequests

import (
	"encoding/json"
	"fmt"
	"master/Driver-go/elevio"
	"master/elevator"
	"master/fsm"
	"master/network/broadcast"
	"master/requests"
	"os/exec"
	"runtime"
	"time"
)

type SlaveButtonEventMsg struct {
	Btn_floor int
	Btn_type  int
}

type MasterAckOrderMsg struct {
	Btn_floor int
	Btn_type  int
}

type HRAElevState struct {
	Behavior    string                 `json:"behaviour"`
	Floor       int                    `json:"floor"`
	Direction   string                 `json:"direction"`
	CabRequests [elevio.NumFloors]bool `json:"cabRequests"`
}

type HRAInput struct {
	HallRequests [][2]bool                    `json:"hallRequests"`
	States       map[string]elevator.Elevator `json:"states"`
}

var e1 elevator.Elevator
var MasterRequests requests.AllRequests
var MasterHallRequests requests.MasterHallRequests

func main() {
	e1 := elevator.Elevator{
		Floor:     1, //jalla
		Dirn:      elevio.MotorDirToString(elevio.MD_Stop),
		Behaviour: elevator.EB_Idle,
	}
	if e1.Floor == -1 {
		e1 = fsm.Fsm_onInitBetweenFloors(e1)
	}

	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "hall_request_assigner"
	case "windows":
		hraExecutable = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}

	//Using elevatorstate as input, HallRequests need to be replaced with MasterRequests
	MasterStruct := HRAInput{
		HallRequests: [][2]bool{{false, false}, {true, false}, {false, false}, {false, true}},
		States: map[string]elevator.Elevator{
			"one": e1,
		},
	}

	input := MasterStruct

	jsonBytes, err := json.Marshal(input)
	fmt.Println("json.Marshal error: ", err)

	ret, err := exec.Command("hall_request_assigner/"+hraExecutable, "-i", string(jsonBytes)).Output()
	fmt.Println("exec.Command error: ", err)

	output := new(map[string][][2]bool)
	err = json.Unmarshal(ret, &output)
	fmt.Println("json.Unmarshal error: ", err)

	fmt.Printf("output: \n")
	for k, v := range *output {
		fmt.Printf("%6v :  %+v\n", k, v)
	}

	fmt.Println(output)
	AssignedRequests := output
	fmt.Println(AssignedRequests)

	slaveButtonRx := make(chan SlaveButtonEventMsg)
	slaveFloorRx := make(chan int)
	masterCommandMD := make(chan string)
	masterAckOrder := make(chan MasterAckOrderMsg)
	slaveAckOrderDoneRx := make(chan bool)
	masterTurnOffOrderLightTx := make(chan int)
	slaveState := make(chan elevator.Elevator)
	commandDoorOpen := make(chan bool)
	slaveDoorOpened := make(chan bool)

	go broadcast.Receiver(16513, slaveButtonRx)
	go broadcast.Receiver(16514, slaveFloorRx)
	go broadcast.Receiver(16517, slaveAckOrderDoneRx)
	go broadcast.Receiver(16519, slaveState)
	go broadcast.Receiver(16521, slaveDoorOpened)

	go broadcast.Transmitter(16515, masterCommandMD)
	go broadcast.Transmitter(16516, masterAckOrder)
	go broadcast.Transmitter(16518, masterTurnOffOrderLightTx)
	go broadcast.Transmitter(16520, commandDoorOpen)

	doorTimer := time.NewTimer(20 * time.Second) //Trouble initializing timer like this, maybe

	for {
		select {
		case slaveMsg := <-slaveButtonRx:
			fmt.Println("Floor")
			fmt.Println(slaveMsg.Btn_floor)
			fmt.Println("Button type")
			fmt.Println(slaveMsg.Btn_type)
			fmt.Println(" ")

			MasterRequests.Requests[slaveMsg.Btn_floor][slaveMsg.Btn_type] = true
			fmt.Println(MasterRequests)

			if slaveMsg.Btn_type != 2 {
				MasterHallRequests.Requests[slaveMsg.Btn_floor][slaveMsg.Btn_type] = true //cant include cab
			}
			fmt.Println(MasterHallRequests)
			if e1.Behaviour == elevator.EB_Idle {
				nextAction := requests.RequestsNextAction(e1, MasterRequests)
				e1.Behaviour = nextAction.Behaviour
				e1.Dirn = elevio.MotorDirToString(nextAction.Dirn)
				masterCommandMD <- e1.Dirn
			}

			masterAckOrder <- MasterAckOrderMsg{int(slaveMsg.Btn_floor), slaveMsg.Btn_type}

		case slaveMsg := <-slaveFloorRx:
			e1.Floor = slaveMsg
			fmt.Println("Behaviour")
			fmt.Println(e1.Behaviour)
			fmt.Println("direction")
			fmt.Println(e1.Dirn)
			a := requests.RequestsNextAction(e1, MasterRequests)
			e1.Behaviour = a.Behaviour
			e1.Dirn = elevio.MotorDirToString(a.Dirn)
			if e1.Behaviour == elevator.EB_DoorOpen {
				commandDoorOpen <- true
			} else {
				masterCommandMD <- e1.Dirn
			}

		case slaveMsg := <-slaveDoorOpened:
			if slaveMsg {
				e1.Behaviour = elevator.EB_DoorOpen
				MasterRequests.Requests[e1.Floor][0] = false
				MasterRequests.Requests[e1.Floor][1] = false
				MasterRequests.Requests[e1.Floor][2] = false

				MasterHallRequests.Requests[e1.Floor][0] = false
				MasterHallRequests.Requests[e1.Floor][1] = false

				fmt.Println(MasterHallRequests)

				masterTurnOffOrderLightTx <- e1.Floor
				doorTimer.Stop()
				doorTimer.Reset(3 * time.Second)
			} else {
				a := requests.RequestsNextAction(e1, MasterRequests)
				e1.Dirn = elevio.MotorDirToString(a.Dirn)
				e1.Behaviour = a.Behaviour

				switch e1.Behaviour {
				case elevator.EB_DoorOpen:
					e1, MasterRequests = requests.ClearRequestCurrentFloor(e1, MasterRequests)
					masterTurnOffOrderLightTx <- e1.Floor
				case elevator.EB_Moving:
					masterCommandMD <- e1.Dirn
				case elevator.EB_Idle:
					masterCommandMD <- e1.Dirn
				}
			}
		case <-doorTimer.C:
			commandDoorOpen <- false
			e1.Behaviour = elevator.EB_Idle
		}
	}

}
