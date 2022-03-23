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
	"master/network/peers"
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
		e2 = fsm.Fsm_onInitBetweenFloors(e2)
	}

	//PeerList := peers.PeerUpdate{Peers: "one", New: "", Lost: ""}
	var PeerList peers.PeerUpdate
	PeerList.Peers = append(PeerList.Peers, "one", "two")

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
		//Add peer list
		HallRequests: [][2]bool{{false, false}, {false, false}, {false, false}, {false, false}},
		States: map[string]elevator.Elevator{
			"one": e1,
			"two": e2,
		},
	}

	input := MasterStruct

	jsonBytes, err := json.Marshal(input)
	fmt.Println("json.Marshal error: ", err)

	ret, err := exec.Command("hall_request_assigner/"+hraExecutable, "-i", string(jsonBytes), "--includeCab").Output()
	fmt.Println("exec.Command error: ", err)

	output := new(map[string][][3]bool)
	err = json.Unmarshal(ret, &output)
	fmt.Println("json.Unmarshal error: ", err)

	fmt.Printf("output: \n")
	for k, v := range *output {
		fmt.Printf("%6v :  %+v\n", k, v)
	}

	slaveButtonRx := make(chan types.SlaveButtonEventMsg)
	slaveFloorRx := make(chan types.SlaveFloor)
	masterCommandMD := make(chan types.MasterCommand)
	//slaveAckOrderDoneRx := make(chan bool)
	masterSetOrderLight := make(chan types.SetOrderLight)
	commandDoorOpen := make(chan types.DoorOpen)
	slaveDoorOpened := make(chan types.DoorOpen)
	NewEvent := make(chan types.HRAInput)
	NewAction := make(chan types.NewAction)

	go broadcast.Receiver(16513, slaveButtonRx)
	go broadcast.Receiver(16514, slaveFloorRx)
	//go broadcast.Receiver(16517, slaveAckOrderDoneRx)
	go broadcast.Receiver(16521, slaveDoorOpened)

	go broadcast.Transmitter(16515, masterCommandMD)
	go broadcast.Transmitter(16518, masterSetOrderLight)
	go broadcast.Transmitter(16520, commandDoorOpen)
	go master.MasterFindNextAction(NewEvent, NewAction, commandDoorOpen, masterCommandMD, PeerList)

	//doorTimer := time.NewTimer(20 * time.Second) //Trouble initializing timer like this, maybe

	for {
		select {
		case slaveMsg := <-slaveButtonRx:
			if slaveMsg.Btn_type == 2 {
				if entry, ok := MasterStruct.States[slaveMsg.ID]; ok {
					entry.CabRequests[slaveMsg.Btn_floor] = true
					MasterStruct.States[slaveMsg.ID] = entry
				}

			} else {
				MasterStruct.HallRequests[slaveMsg.Btn_floor][slaveMsg.Btn_type] = true
			}
			NewEvent <- MasterStruct
			SetLightArray := [3]bool{
				MasterStruct.HallRequests[slaveMsg.Btn_floor][elevio.BT_HallUp],
				MasterStruct.HallRequests[slaveMsg.Btn_floor][elevio.BT_HallDown],
				MasterStruct.States[slaveMsg.ID].CabRequests[slaveMsg.Btn_floor]}
			SetLightArray[slaveMsg.Btn_type] = true
			SetOrderLight := types.SetOrderLight{ID: slaveMsg.ID, BtnFloor: slaveMsg.Btn_floor, LightOn: SetLightArray}
			masterSetOrderLight <- SetOrderLight

		case slaveMsg := <-slaveFloorRx:
			elevatorID := slaveMsg.ID
			elevatorFloor := slaveMsg.NewFloor

			if entry, ok := MasterStruct.States[elevatorID]; ok {
				entry.Floor = elevatorFloor
				entry.Behaviour = elevator.EB_Idle
				MasterStruct.States[elevatorID] = entry
			}
			NewEvent <- MasterStruct

		case slaveMsg := <-slaveDoorOpened:
			elevState := MasterStruct.States[slaveMsg.ID]
			if slaveMsg.SetDoorOpen {
				if entry, ok := MasterStruct.States[slaveMsg.ID]; ok {
					entry.Behaviour = elevator.EB_DoorOpen
					entry.CabRequests[elevState.Floor] = false
					MasterStruct.States[slaveMsg.ID] = entry
				}
				elevState = MasterStruct.States[slaveMsg.ID]

				ClearHallReqs := requests.ShouldClearHallRequest(elevState, MasterStruct.HallRequests)
				fmt.Println()
				MasterStruct.HallRequests[elevState.Floor][elevio.BT_HallUp] = ClearHallReqs[elevio.BT_HallUp]
				MasterStruct.HallRequests[elevState.Floor][elevio.BT_HallDown] = ClearHallReqs[elevio.BT_HallDown]

				ClearLightArray := [3]bool{ClearHallReqs[elevio.BT_HallUp], ClearHallReqs[elevio.BT_HallDown], false}
				SetOrderLight := types.SetOrderLight{ID: slaveMsg.ID, BtnFloor: elevState.Floor, LightOn: ClearLightArray}
				masterSetOrderLight <- SetOrderLight
			}
			NewEvent <- MasterStruct

		case a := <-NewAction:
			if entry, ok := MasterStruct.States[a.ID]; ok {
				entry.Behaviour = a.Action.Behaviour
				entry.Dirn = elevio.MotorDirToString(a.Action.Dirn)
				MasterStruct.States[a.ID] = entry
			}
			fmt.Println(a.ID)
			fmt.Println(MasterStruct.States[a.ID].Behaviour)
			fmt.Println(MasterStruct.States[a.ID].Dirn)
			if MasterStruct.States[a.ID].Behaviour == elevator.EB_DoorOpen {
				commandDoorOpen <- types.DoorOpen{ID: a.ID, SetDoorOpen: true}
			} else {
				masterCommandMD <- types.MasterCommand{ID: a.ID, Motordir: elevio.MotorDirToString(a.Action.Dirn)}
			}

		}
	}
}
