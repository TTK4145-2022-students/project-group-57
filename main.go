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
	"master/types"
	"os/exec"
	"runtime"
	"time"
)

var e1 elevator.Elevator
var MasterRequests types.AllRequests
var MasterHallRequests types.MasterHallRequests

func main() {
	MasterHallRequests.Requests = [elevio.NumFloors][2]bool{}
	fmt.Println(MasterHallRequests)
	e1 := elevator.Elevator{
		Floor:       1, //jalla
		Dirn:        elevio.MotorDirToString(elevio.MD_Stop),
		Behaviour:   elevator.EB_Idle,
		CabRequests: [elevio.NumFloors]bool{},
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
	MasterStruct := types.HRAInput{
		HallRequests: [elevio.NumFloors][2]bool{{false, false}, {true, false}, {false, false}, {false, true}},
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

	slaveButtonRx := make(chan types.SlaveButtonEventMsg)
	slaveFloorRx := make(chan types.SlaveFloor)
	masterCommandMD := make(chan string)
	masterAckOrder := make(chan types.MasterAckOrderMsg)
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
			if slaveMsg.Btn_type == 2 {
				e1.CabRequests[slaveMsg.Btn_floor] = true
			} else {
				MasterHallRequests.Requests[slaveMsg.Btn_floor][slaveMsg.Btn_type] = true
			}
			MasterStruct.HallRequests = MasterHallRequests.Requests
			MasterStruct.States["one"] = e1
			fmt.Println("Hallrequests: ")
			fmt.Println(MasterHallRequests)
			fmt.Println("Cabrequests: ")
			fmt.Println(e1.CabRequests)
			fmt.Println("Masterstruct: ")
			fmt.Println(MasterStruct)

			fmt.Println(e1.Behaviour)
			fmt.Println(e1.Dirn)

			if e1.Behaviour == "idle" {
				fmt.Println("inside if")
				nextAction := requests.RequestsNextAction(e1, MasterHallRequests)
				e1.Behaviour = nextAction.Behaviour
				e1.Dirn = elevio.MotorDirToString(nextAction.Dirn)
				fmt.Println(e1.Behaviour)
				fmt.Println(e1.Dirn)
				masterCommandMD <- e1.Dirn
			}

			masterAckOrder <- types.MasterAckOrderMsg{int(slaveMsg.Btn_floor), slaveMsg.Btn_type}

		case slaveMsg := <-slaveFloorRx:
			//send to correct ID
			elevatorID := slaveMsg.ID
			elevatorFloor := slaveMsg.NewFloor
			fmt.Println(elevatorID)
			fmt.Println(elevatorFloor)
			fmt.Println(e1.Behaviour)
			fmt.Println("direction")
			fmt.Println(e1.Dirn)

			MasterStruct.States["one"] = e1

			/*input := MasterStruct

			jsonBytes, err := json.Marshal(input)
			fmt.Println("json.Marshal error: ", err)

			ret, err := exec.Command("hall_request_assigner/"+hraExecutable, "-i", string(jsonBytes)).Output()
			fmt.Println("exec.Command error: ", err)

			output = new(map[string][][2]bool)
			err = json.Unmarshal(ret, &output)
			fmt.Println("json.Unmarshal error: ", err)
			fmt.Println(*output)
			for _, v := range *output {
				fmt.Printf("%+v\n", v)
				fmt.Println(v[1])
				HallReqs := v
				//requests.RequestsNextAction()
				fmt.Println("Hallreqs: ")
				fmt.Println(HallReqs)
			}*/
			//b := requests.RequestsNextAction(MasterStruct.States["one"], *output) //Like this
			//fmt.Println(b)

			a := requests.RequestsNextAction(e1, MasterHallRequests)
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
				/*MasterRequests.Requests[e1.Floor][0] = false
				MasterRequests.Requests[e1.Floor][1] = false
				MasterRequests.Requests[e1.Floor][2] = false
				*/

				MasterHallRequests.Requests[e1.Floor][0] = false
				MasterHallRequests.Requests[e1.Floor][1] = false

				fmt.Println(MasterHallRequests)

				masterTurnOffOrderLightTx <- e1.Floor
				doorTimer.Stop()
				doorTimer.Reset(3 * time.Second)
			} else {
				a := requests.RequestsNextAction(e1, MasterHallRequests)
				e1.Dirn = elevio.MotorDirToString(a.Dirn)
				e1.Behaviour = a.Behaviour
				fmt.Println("New action received")
				fmt.Println(e1.Behaviour)

				switch e1.Behaviour {
				case elevator.EB_DoorOpen:
					e1, MasterHallRequests = requests.ClearRequestCurrentFloor(e1, MasterHallRequests)
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

//Reassigning orders at each new action/event?
//How to request next action based on which elevator?:
//Each channel includes which elevator it comes from?
//-cons many channels (maybe necessary?)
//Not sending MasterRequests as input to request-funcs
//Send rather corresponding request-array from hra-output
