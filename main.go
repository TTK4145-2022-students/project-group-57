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
	"master/master"
	"master/network/broadcast"
	"master/requests"
	"master/types"
	"os/exec"
	"runtime"
)

var e1 elevator.Elevator
var MasterRequests types.AllRequests
var MasterHallRequests types.MasterHallRequests

func main() {
	e1 := elevator.Elevator{
		Floor:       1, //jalla
		Dirn:        elevio.MotorDirToString(elevio.MD_Stop),
		Behaviour:   elevator.EB_Idle,
		CabRequests: [elevio.NumFloors]bool{},
	}
	e2 := elevator.Elevator{
		Floor:       1, //jalla
		Dirn:        elevio.MotorDirToString(elevio.MD_Stop),
		Behaviour:   elevator.EB_Idle,
		CabRequests: [elevio.NumFloors]bool{},
	}
	if e1.Floor == -1 {
		e1 = fsm.Fsm_onInitBetweenFloors(e1)
	}
	if e2.Floor == -1 {
		e1 = fsm.Fsm_onInitBetweenFloors(e2)
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
		HallRequests: [][2]bool{{false, false}, {false, false}, {false, false}, {false, false}},
		States: map[string]elevator.Elevator{
			"one": e1,
			"two": e2,
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
	masterCommandMD := make(chan types.MasterCommand)
	masterAckOrder := make(chan types.MasterAckOrderMsg)
	//slaveAckOrderDoneRx := make(chan bool)
	masterSetOrderLight := make(chan int)
	commandDoorOpen := make(chan types.DoorOpen)
	slaveDoorOpened := make(chan types.DoorOpen)
	NewEvent := make(chan types.HRAInput)
	NewAction := make(chan requests.Action)

	go broadcast.Receiver(16513, slaveButtonRx)
	go broadcast.Receiver(16514, slaveFloorRx)
	//go broadcast.Receiver(16517, slaveAckOrderDoneRx)
	go broadcast.Receiver(16521, slaveDoorOpened)

	go broadcast.Transmitter(16515, masterCommandMD)
	go broadcast.Transmitter(16516, masterAckOrder)
	go broadcast.Transmitter(16518, masterSetOrderLight)
	go broadcast.Transmitter(16520, commandDoorOpen)

	go master.MasterGiveCommands(NewEvent, NewAction, commandDoorOpen, masterCommandMD)

	//doorTimer := time.NewTimer(20 * time.Second) //Trouble initializing timer like this, maybe

	for {
		select {
		case slaveMsg := <-slaveButtonRx:
			if slaveMsg.Btn_type == 2 {
				e1.CabRequests[slaveMsg.Btn_floor] = true
			} else {
				MasterStruct.HallRequests[slaveMsg.Btn_floor][slaveMsg.Btn_type] = true
			}
			MasterStruct.States["one"] = e1 //fix
			NewEvent <- MasterStruct

		case slaveMsg := <-slaveFloorRx:
			elevatorID := slaveMsg.ID
			elevatorFloor := slaveMsg.NewFloor

			if entry, ok := MasterStruct.States[elevatorID]; ok {
				entry.Floor = elevatorFloor
				MasterStruct.States[elevatorID] = entry
			}
			NewEvent <- MasterStruct

		case slaveMsg := <-slaveDoorOpened:
			if slaveMsg.SetDoorOpen {
				if entry, ok := MasterStruct.States[slaveMsg.ID]; ok {
					entry.Behaviour = elevator.EB_DoorOpen
				}

				elevState := MasterStruct.States[slaveMsg.ID]
				MasterStruct.HallRequests[elevState.Floor][0] = false
				MasterStruct.HallRequests[elevState.Floor][1] = false

				NewEvent <- MasterStruct
			} else {
				NewEvent <- MasterStruct
			}
		case a := <-NewAction:
			if entry, ok := MasterStruct.States["one"]; ok {
				entry.Behaviour = a.Behaviour
				entry.Dirn = elevio.MotorDirToString(a.Dirn)
				MasterStruct.States["one"] = entry
			}

		}
	}

	/*for {
			select {
			case slaveMsg := <-slaveButtonRx:
				if slaveMsg.Btn_type == 2 {
					e1.CabRequests[slaveMsg.Btn_floor] = true
				} else {
					MasterStruct.HallRequests[slaveMsg.Btn_floor][slaveMsg.Btn_type] = true
				}
				MasterStruct.States["one"] = e1

				/*if e1.Behaviour == "idle" {
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

				if entry, ok := MasterStruct.States[elevatorID]; ok {
					entry.Floor = elevatorFloor
					MasterStruct.States[elevatorID] = entry
				}

				input := MasterStruct
				fmt.Println(input)

				jsonBytes, err := json.Marshal(input)
				fmt.Println("json.Marshal error: ", err)

				ret, err := exec.Command("hall_request_assigner/"+hraExecutable, "-i", string(jsonBytes)).Output()
				fmt.Println("exec.Command error: ", err)

				err = json.Unmarshal(ret, &output)
				fmt.Println("json.Unmarshal error: ", err)

				fmt.Printf("output: \n")
				for k, v := range *output {
					fmt.Printf("%6v :  %+v\n", k, v)

				}

				ElevatorHallReqs := (*output)[elevatorID]
				fmt.Println(ElevatorHallReqs)
				elevState := MasterStruct.States[slaveMsg.ID]

				a := requests.RequestsNextAction(elevState, ElevatorHallReqs)
				if entry, ok := MasterStruct.States[elevatorID]; ok {
					entry.Behaviour = a.Behaviour
					entry.Dirn = elevio.MotorDirToString(a.Dirn)
					MasterStruct.States[elevatorID] = entry
				}

				if MasterStruct.States[elevatorID].Behaviour == elevator.EB_DoorOpen {
					commandDoorOpen <- types.DoorOpen{elevatorID, true}
				} else {
					masterCommandMD <- types.MasterCommand{elevatorID, elevio.MotorDirToString(a.Dirn)}
				}

			case slaveMsg := <-slaveDoorOpened:
				if slaveMsg.SetDoorOpen {
					if entry, ok := MasterStruct.States[slaveMsg.ID]; ok {
						entry.Behaviour = elevator.EB_DoorOpen
					}

					elevState := MasterStruct.States[slaveMsg.ID]
					MasterStruct.HallRequests[elevState.Floor][0] = false
					MasterStruct.HallRequests[elevState.Floor][1] = false

					masterSetOrderLight <- elevState.Floor
					doorTimer.Stop()
					doorTimer.Reset(3 * time.Second)
				} else {

					input := MasterStruct

					fmt.Println(input)

					jsonBytes, err := json.Marshal(input)
					fmt.Println("json.Marshal error: ", err)

					ret, err := exec.Command("hall_request_assigner/"+hraExecutable, "-i", string(jsonBytes)).Output()
					fmt.Println("exec.Command error: ", err)

					output = new(map[string][][2]bool)
					err = json.Unmarshal(ret, &output)
					fmt.Println("json.Unmarshal error: ", err)

					fmt.Printf("output: \n")
					for k, v := range *output {
						fmt.Printf("%6v :  %+v\n", k, v)

					}

					ElevatorHallReqs := (*output)[slaveMsg.ID]
					elevState := MasterStruct.States[slaveMsg.ID]

					a := requests.RequestsNextAction(elevState, ElevatorHallReqs)
					if entry, ok := MasterStruct.States[slaveMsg.ID]; ok {
						entry.Behaviour = a.Behaviour
						entry.Dirn = elevio.MotorDirToString(a.Dirn)
						MasterStruct.States[slaveMsg.ID] = entry
					}

					switch elevState.Behaviour {
					case elevator.EB_DoorOpen:
						elevState, MasterStruct.HallRequests = requests.ClearRequestCurrentFloor(elevState, ElevatorHallReqs)
						masterSetOrderLight <- e1.Floor
					case elevator.EB_Moving:
						masterCommandMD <- types.MasterCommand{slaveMsg.ID, elevio.MotorDirToString(a.Dirn)}
					case elevator.EB_Idle:
						masterCommandMD <- types.MasterCommand{slaveMsg.ID, elevio.MotorDirToString(a.Dirn)}
					}
				}
			case <-doorTimer.C:
				commandDoorOpen <- types.DoorOpen{"one", false}
				commandDoorOpen <- types.DoorOpen{"two", false}

				if entry, ok := MasterStruct.States["one"]; ok {
					entry.Behaviour = elevator.EB_Idle
					MasterStruct.States["one"] = entry
				}

				if entry, ok := MasterStruct.States["two"]; ok {
					entry.Behaviour = elevator.EB_Idle
					MasterStruct.States["two"] = entry
				}
			}
		}

	}
	*/
	//Reassigning orders at each new action/event?
	//How to request next action based on which elevator?:
	//Each channel includes which elevator it comes from?
	//-cons many channels (maybe necessary?)
	//Not sending MasterRequests as input to request-funcs
	//Send rather corresponding request-array from hra-output
}
