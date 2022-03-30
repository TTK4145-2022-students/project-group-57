package main

//Problems:
//HallUp and HallDown in current floor, Cab in diff floor after HallUp is executed, but before HallDown is executed
//---- Elevator doesn't execute cab before HallDown
//Concurrent map read and write

import (
	"fmt"
	"master/Driver-go/elevio"
	"master/elevator"
	"master/fsm"
	"master/master"
	"master/network/broadcast"
	"master/network/peers"
	"master/requests"
	"master/types"
	"time"
)

var e1 elevator.Elevator
var MasterRequests types.AllRequests
var MasterHallRequests types.MasterHallRequests

func main() {

	slaveButtonRx := make(chan types.SlaveButtonEventMsg)
	slaveFloorRx := make(chan types.SlaveFloor, 5)
	masterCommandMD := make(chan types.MasterCommand)
	//slaveAckOrderDoneRx := make(chan bool)
	masterSetOrderLight := make(chan types.SetOrderLight)
	commandDoorOpen := make(chan types.DoorOpen)
	slaveDoorOpened := make(chan types.DoorOpen)
	NewEvent := make(chan types.MasterStruct, 5) //Can only handle two button presses at the same time
	NewAction := make(chan types.NewAction, 5)
	PeerUpdateCh := make(chan peers.PeerUpdate)
	MasterMsg := make(chan types.MasterStruct, 3)
	MasterInitStruct := make(chan types.MasterStruct)

	go broadcast.Receiver(16513, slaveButtonRx)
	go broadcast.Receiver(16514, slaveFloorRx)
	//go broadcast.Receiver(16517, slaveAckOrderDoneRx)
	go broadcast.Receiver(16521, slaveDoorOpened)
	go broadcast.Receiver(16524, MasterInitStruct)

	go broadcast.Transmitter(16515, masterCommandMD)
	go broadcast.Transmitter(16518, masterSetOrderLight)
	go broadcast.Transmitter(16520, commandDoorOpen)
	go master.MasterFindNextAction(NewEvent, NewAction, commandDoorOpen, masterCommandMD)
	go broadcast.TransmitMasterMsg(16523, MasterMsg)
	//doorTimer := time.NewTimer(20 * time.Second) //Trouble initializing timer like this, maybe

	go peers.Receiver(16522, PeerUpdateCh)

	const interval = 500 * time.Millisecond
	var NewPeerList peers.PeerUpdate

	fmt.Println("Waiting")

	MasterStruct := <-MasterInitStruct

	if MasterStruct.Ready == false {
		fmt.Println("Ready false")

		HRAInput := types.HRAInput{
			HallRequests: [][2]bool{{false, false}, {false, false}, {false, false}, {false, false}},
			States:       map[string]elevator.Elevator{},
		}

		MasterStruct = types.MasterStruct{
			Ready:    true,
			PeerList: NewPeerList,
			HRAInput: HRAInput,
		}

	} else {
		fmt.Println("none")

	}

	fmt.Println("Masterstruct:")
	fmt.Println(MasterStruct)

	MasterStruct.Ready = true
	for {
		select {
		case <-time.After(interval):
			MasterMsg <- MasterStruct
			//Sending periodically to slaves

		case NewPeerList = <-PeerUpdateCh:
			MasterStruct.PeerList = NewPeerList
			fmt.Println("Peers")
			fmt.Println(MasterStruct.PeerList.Peers)
			fmt.Println("Lost")
			fmt.Println(MasterStruct.PeerList.Lost)
			fmt.Println("New")
			fmt.Println(MasterStruct.PeerList.New)
			var e elevator.Elevator
			e = fsm.UnInitializedElevator(e)
			masterCommandMD <- types.MasterCommand{ID: MasterStruct.PeerList.New, Motordir: "down"}
			if _, ok := MasterStruct.HRAInput.States[MasterStruct.PeerList.New]; ok {
			} else if MasterStruct.PeerList.New != "" {
				MasterStruct.HRAInput.States[MasterStruct.PeerList.New] = e //Fix here
			}

			fmt.Println(MasterStruct.HRAInput.States)
			NewEvent <- MasterStruct
			fmt.Println("Sent")

		case slaveMsg := <-slaveButtonRx:
			if slaveMsg.Btn_type == 2 {
				if entry, ok := MasterStruct.HRAInput.States[slaveMsg.ID]; ok {
					entry.CabRequests[slaveMsg.Btn_floor] = true
					MasterStruct.HRAInput.States[slaveMsg.ID] = entry
				}

			} else {
				MasterStruct.HRAInput.HallRequests[slaveMsg.Btn_floor][slaveMsg.Btn_type] = true
			}
			SetLightArray := [3]bool{
				MasterStruct.HRAInput.HallRequests[slaveMsg.Btn_floor][elevio.BT_HallUp],
				MasterStruct.HRAInput.HallRequests[slaveMsg.Btn_floor][elevio.BT_HallDown],
				MasterStruct.HRAInput.States[slaveMsg.ID].CabRequests[slaveMsg.Btn_floor]}
			SetLightArray[slaveMsg.Btn_type] = true
			SetOrderLight := types.SetOrderLight{ID: slaveMsg.ID, BtnFloor: slaveMsg.Btn_floor, LightOn: SetLightArray}

			NewEvent <- MasterStruct
			masterSetOrderLight <- SetOrderLight

		case slaveMsg := <-slaveFloorRx:
			elevatorID := slaveMsg.ID
			elevatorFloor := slaveMsg.NewFloor

			if entry, ok := MasterStruct.HRAInput.States[elevatorID]; ok {
				entry.Floor = elevatorFloor
				entry.Behaviour = elevator.EB_Idle
				MasterStruct.HRAInput.States[elevatorID] = entry
			}
			NewEvent <- MasterStruct

		case slaveMsg := <-slaveDoorOpened:
			elevState := MasterStruct.HRAInput.States[slaveMsg.ID]
			if slaveMsg.SetDoorOpen {
				if entry, ok := MasterStruct.HRAInput.States[slaveMsg.ID]; ok {
					entry.Behaviour = elevator.EB_DoorOpen
					entry.CabRequests[elevState.Floor] = false
					MasterStruct.HRAInput.States[slaveMsg.ID] = entry
				}
				elevState = MasterStruct.HRAInput.States[slaveMsg.ID]

				ClearHallReqs := requests.ShouldClearHallRequest(elevState, MasterStruct.HRAInput.HallRequests)
				fmt.Println()
				MasterStruct.HRAInput.HallRequests[elevState.Floor][elevio.BT_HallUp] = ClearHallReqs[elevio.BT_HallUp]
				MasterStruct.HRAInput.HallRequests[elevState.Floor][elevio.BT_HallDown] = ClearHallReqs[elevio.BT_HallDown]

				ClearLightArray := [3]bool{ClearHallReqs[elevio.BT_HallUp], ClearHallReqs[elevio.BT_HallDown], false}
				SetOrderLight := types.SetOrderLight{ID: slaveMsg.ID, BtnFloor: elevState.Floor, LightOn: ClearLightArray}
				masterSetOrderLight <- SetOrderLight
			}
			NewEvent <- MasterStruct

		case a := <-NewAction:
			if entry, ok := MasterStruct.HRAInput.States[a.ID]; ok {
				entry.Behaviour = a.Action.Behaviour
				entry.Dirn = elevio.MotorDirToString(a.Action.Dirn)
				MasterStruct.HRAInput.States[a.ID] = entry
			}
			fmt.Println(a.ID)
			fmt.Println(MasterStruct.HRAInput.States[a.ID].Behaviour)
			fmt.Println(MasterStruct.HRAInput.States[a.ID].Dirn)
			if MasterStruct.HRAInput.States[a.ID].Behaviour == elevator.EB_DoorOpen {
				commandDoorOpen <- types.DoorOpen{ID: a.ID, SetDoorOpen: true}
			} else {
				masterCommandMD <- types.MasterCommand{ID: a.ID, Motordir: elevio.MotorDirToString(a.Action.Dirn)}
			}

		}
	}
}
