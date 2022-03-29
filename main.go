package main

//Problems:
//HallUp and HallDown in current floor, Cab in diff floor after HallUp is executed, but before HallDown is executed
//---- Elevator doesn't execute cab before HallDown
//Concurrent map read and write

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
	"time"
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
	e3 := elevator.Elevator{
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
	if e3.Floor == -1 {
		e3 = fsm.Fsm_onInitBetweenFloors(e3)
	}

	//PeerList := peers.PeerUpdate{Peers: "one", New: "", Lost: ""}
	var PeerList peers.PeerUpdate
	//PeerList.Peers = append(PeerList.Peers, "one", "two", "three")
	PeerList.Peers = append(PeerList.Peers, "one")
	var NewPeerList peers.PeerUpdate

	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "hall_request_assigner"
	case "windows":
		hraExecutable = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}

	/*
		MasterStruct := types.HRAInput{
			//Add peer list
			HallRequests: [][2]bool{{false, false}, {false, false}, {false, false}, {false, false}},
			States: map[string]elevator.Elevator{
				"one":   e1,
				"two":   e2,
				"three": e3,
			},
		}
	*/

	MasterStruct := types.HRAInput{
		//Add peer list
		HallRequests: [][2]bool{{false, false}, {false, false}, {false, false}, {false, false}},
		States:       map[string]elevator.Elevator{},
	}

	input := MasterStruct
	const interval = 500 * time.Millisecond

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
	NewEvent := make(chan types.HRAInput, 1) //Can only handle two button presses at the same time
	NewAction := make(chan types.NewAction, 1)
	PeerUpdateCh := make(chan peers.PeerUpdate)
	NewPeerListCh := make(chan peers.PeerUpdate)
	MasterMsg := make(chan types.HRAInput, 3)

	go broadcast.Receiver(16513, slaveButtonRx)
	go broadcast.Receiver(16514, slaveFloorRx)
	//go broadcast.Receiver(16517, slaveAckOrderDoneRx)
	go broadcast.Receiver(16521, slaveDoorOpened)

	go broadcast.Transmitter(16515, masterCommandMD)
	go broadcast.Transmitter(16518, masterSetOrderLight)
	go broadcast.Transmitter(16520, commandDoorOpen)
	go master.MasterFindNextAction(NewEvent, NewAction, commandDoorOpen, masterCommandMD, NewPeerListCh, MasterMsg)
	go broadcast.TransmitMasterMsg(16523, MasterMsg)
	//doorTimer := time.NewTimer(20 * time.Second) //Trouble initializing timer like this, maybe

	go peers.Receiver(16522, PeerUpdateCh)

	for {
		select {
		case <-time.After(interval):
			MasterMsg <- MasterStruct
			//Sending periodically to slaves

		case NewPeerList = <-PeerUpdateCh:
			fmt.Println("Peers")
			fmt.Println(NewPeerList.Peers)
			fmt.Println("Lost")
			fmt.Println(NewPeerList.Lost)
			fmt.Println("New")
			fmt.Println(NewPeerList.New)
			for j := range NewPeerList.New {
				fmt.Println(j)
				var e elevator.Elevator
				e = fsm.UnInitializedElevator(e)
				MasterStruct.States[NewPeerList.New] = e //Fix here
			}
			for k := range NewPeerList.Lost {
				delete(MasterStruct.States, NewPeerList.Lost[k])
			}
			fmt.Println(MasterStruct.States)

		case slaveMsg := <-slaveButtonRx:
			if slaveMsg.Btn_type == 2 {
				if entry, ok := MasterStruct.States[slaveMsg.ID]; ok {
					entry.CabRequests[slaveMsg.Btn_floor] = true
					MasterStruct.States[slaveMsg.ID] = entry
				}

			} else {
				MasterStruct.HallRequests[slaveMsg.Btn_floor][slaveMsg.Btn_type] = true
			}
			SetLightArray := [3]bool{
				MasterStruct.HallRequests[slaveMsg.Btn_floor][elevio.BT_HallUp],
				MasterStruct.HallRequests[slaveMsg.Btn_floor][elevio.BT_HallDown],
				MasterStruct.States[slaveMsg.ID].CabRequests[slaveMsg.Btn_floor]}
			SetLightArray[slaveMsg.Btn_type] = true
			SetOrderLight := types.SetOrderLight{ID: slaveMsg.ID, BtnFloor: slaveMsg.Btn_floor, LightOn: SetLightArray}

			NewEvent <- MasterStruct
			masterSetOrderLight <- SetOrderLight
			NewPeerListCh <- NewPeerList

		case slaveMsg := <-slaveFloorRx:
			elevatorID := slaveMsg.ID
			elevatorFloor := slaveMsg.NewFloor

			if entry, ok := MasterStruct.States[elevatorID]; ok {
				entry.Floor = elevatorFloor
				entry.Behaviour = elevator.EB_Idle
				MasterStruct.States[elevatorID] = entry
			}
			NewEvent <- MasterStruct
			NewPeerListCh <- NewPeerList

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
			NewPeerListCh <- NewPeerList

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
