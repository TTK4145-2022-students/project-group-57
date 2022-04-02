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
	"os"
	"strconv"
	"time"
)

var MasterRequests types.AllRequests
var MasterHallRequests types.MasterHallRequests

func main() {

	slaveButtonRx := make(chan types.SlaveButtonEventMsg)
	slaveFloorRx := make(chan types.SlaveFloor, 5)
	masterCommandMD := make(chan types.MasterCommand)
	masterSetOrderLight := make(chan types.SetOrderLight)
	commandDoorOpen := make(chan types.DoorOpen)
	slaveDoorOpened := make(chan types.DoorOpen)
	NewEvent := make(chan types.MasterStruct, 5)
	NewAction := make(chan types.NewAction, 5)
	PeerUpdateCh := make(chan peers.PeerUpdate)
	MasterMsg := make(chan types.MasterStruct, 3)
	MasterInitStruct := make(chan types.MasterStruct)
	MasterMergeSend := make(chan types.MasterStruct, 3)
	MasterMergeReceive := make(chan types.MasterStruct, 3)

	NewMasterIDCh := make(chan types.NewMasterID)

	go broadcast.Receiver(16513, slaveButtonRx)
	go broadcast.Receiver(16514, slaveFloorRx)
	go broadcast.Receiver(16521, slaveDoorOpened)
	go broadcast.Receiver(16527, MasterInitStruct)
	go broadcast.Transmitter(16515, masterCommandMD)
	go broadcast.Transmitter(16518, masterSetOrderLight)
	go broadcast.Transmitter(16520, commandDoorOpen)
	go master.MasterFindNextAction(NewEvent, NewAction, commandDoorOpen, masterCommandMD)
	go broadcast.TransmitMasterMsg(16523, MasterMsg)
	go broadcast.Transmitter(16524, NewMasterIDCh)
	go broadcast.Transmitter(16585, MasterMergeSend)
	go broadcast.Receiver(16585, MasterMergeReceive)

	go peers.Receiver(16522, PeerUpdateCh)

	const interval = 500 * time.Millisecond
	var NewPeerList peers.PeerUpdate

	initArg := os.Args[1]

	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println(initArg)

	var MasterStruct types.MasterStruct
	MasterStruct.HRAInput.States = map[string]elevator.Elevator{}
	MasterStruct.AlreadyExists = false
	if initArg == "init" {
		SlaveID := os.Args[2]
		CurrentFloor := os.Args[3]
		fmt.Println(SlaveID)
		fmt.Println(CurrentFloor)
		fmt.Println("fake master")
		var e elevator.Elevator
		e = fsm.UnInitializedElevator(e)
		MasterStruct.HRAInput.States[SlaveID] = e
		fmt.Println(MasterStruct)
		if entry, ok := MasterStruct.HRAInput.States[SlaveID]; ok {
			fmt.Print("entry")
			entry.Floor, _ = strconv.Atoi(CurrentFloor)
			MasterStruct.HRAInput.States[SlaveID] = entry
			fmt.Println(MasterStruct)
		}
		//Sends to already existing master
		//Send for a longer time?
		//
		for i := 0; i < 5; i++ {
			MasterMergeSend <- MasterStruct
			time.Sleep(300 * time.Millisecond)
		}
		fmt.Println("Time to kill")
		os.Exit(99)
	}

	MasterStruct = <-MasterInitStruct
	MasterStruct.MySlaves = []string{MasterStruct.CurrentMasterID}

	if !MasterStruct.AlreadyExists {
		//Check out other possibilities, can be removed if we find another way to check if empty struct

		HRAInput := types.HRAInput{
			HallRequests: [][2]bool{{false, false}, {false, false}, {false, false}, {false, false}},
			States:       MasterStruct.HRAInput.States,
		}

		MasterStruct = types.MasterStruct{
			CurrentMasterID: MasterStruct.CurrentMasterID,
			Isolated:        false,
			AlreadyExists:   true,
			PeerList:        NewPeerList,
			HRAInput:        HRAInput,
			MySlaves:        []string{MasterStruct.CurrentMasterID},
		}

	}

	MasterStruct.AlreadyExists = true
	for {
		select {
		case <-time.After(interval):
			MasterMsg <- MasterStruct

		case ReceivedMergeStruct := <-MasterMergeReceive:
			if !ReceivedMergeStruct.AlreadyExists { //Existing master, receiving initstruct from initmaster
				for k := range ReceivedMergeStruct.HRAInput.States { //Single slaveID
					ReceivedID := k
					if entry, ok := ReceivedMergeStruct.HRAInput.States[ReceivedID]; ok { //Keeping floor & cab from receivedStruct
						entry.Floor = ReceivedMergeStruct.HRAInput.States[ReceivedID].Floor
						entry.CabRequests = MasterStruct.HRAInput.States[ReceivedID].CabRequests
						MasterStruct.HRAInput.States[ReceivedID] = entry
						MasterStruct.MySlaves = append(MasterStruct.MySlaves, k) //APPEND WITHOUT DUPLICATES MAKE FUNCTION 
					} //ReceivedID exists in MasterStruct
				}
				HallRequests := MasterStruct.HRAInput.HallRequests
				for k := range MasterStruct.HRAInput.States { //Merge for nonexisting/empty receivedStruct
					CabRequests := MasterStruct.HRAInput.States[k].CabRequests
					AllRequests := requests.RequestsAppendHallCab(HallRequests, CabRequests)
					SetOrderLight := types.SetOrderLight{MasterID: MasterStruct.CurrentMasterID, ID: k, LightOn: AllRequests}
					masterSetOrderLight <- SetOrderLight
				}
			} else {
				if !MasterStruct.Isolated {
					if ReceivedMergeStruct.Isolated {
						NewMasterID := ReceivedMergeStruct.PeerList.Peers[0]
						if NewMasterID == MasterStruct.CurrentMasterID {
							MasterStruct.MySlaves = append(MasterStruct.MySlaves, ReceivedMergeStruct.PeerList.Peers...)
							MasterStruct = master.MergeMasterStructs(MasterStruct, ReceivedMergeStruct)
						} else {
							for i := 0; i < 10; i++ {
								MasterMergeSend <- MasterStruct
								time.Sleep(100 * time.Millisecond)
							}
							os.Exit(3)
						}
					} else {
						MasterStruct.MySlaves = append(MasterStruct.MySlaves, ReceivedMergeStruct.PeerList.Peers...)
						MasterStruct = master.MergeMasterStructs(MasterStruct, ReceivedMergeStruct)
					}
				}
			}
			NewEvent <- MasterStruct

		case NewPeerList = <-PeerUpdateCh: //Use only for deleting, not adding new
			fmt.Println("Peerlist")
			fmt.Println(NewPeerList)
			LostPeers := NewPeerList.Lost
			if len(LostPeers) != 0 {
				var MySlavesCopy = []string{}
				for k := range LostPeers { //MAKE FUNCTION
					for j := range MasterStruct.MySlaves {
						if LostPeers[k] == MasterStruct.MySlaves[j] {
							PeerToRemove := LostPeers[k]
							for i := range MasterStruct.MySlaves {
								if MasterStruct.MySlaves[i] != PeerToRemove {
									MySlavesCopy = append(MySlavesCopy, MasterStruct.MySlaves[i]) //MAKE FUNCTION //APPEND WITHOUT DUPLICATES
								}
							}
						}
					}
				}
				fmt.Println("Peerupdate")
				fmt.Println(MasterStruct.MySlaves)
				MasterStruct.MySlaves = MySlavesCopy
				fmt.Println(MySlavesCopy)
				NewEvent <- MasterStruct
			}

		case slaveMsg := <-slaveButtonRx:
			if slaveMsg.Btn_type == 2 {
				if entry, ok := MasterStruct.HRAInput.States[slaveMsg.ID]; ok {
					entry.CabRequests[slaveMsg.Btn_floor] = true
					MasterStruct.HRAInput.States[slaveMsg.ID] = entry
				}
			} else {
				MasterStruct.HRAInput.HallRequests[slaveMsg.Btn_floor][slaveMsg.Btn_type] = true
			}
			AllRequests := requests.RequestsAppendHallCab(MasterStruct.HRAInput.HallRequests, MasterStruct.HRAInput.States[slaveMsg.ID].CabRequests)
			SetOrderLight := types.SetOrderLight{MasterID: MasterStruct.CurrentMasterID, ID: slaveMsg.ID, LightOn: AllRequests}
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
				MasterStruct.HRAInput.HallRequests[elevState.Floor][elevio.BT_HallUp] = ClearHallReqs[elevio.BT_HallUp]
				MasterStruct.HRAInput.HallRequests[elevState.Floor][elevio.BT_HallDown] = ClearHallReqs[elevio.BT_HallDown]
				AllRequests := requests.RequestsAppendHallCab(MasterStruct.HRAInput.HallRequests, MasterStruct.HRAInput.States[slaveMsg.ID].CabRequests)
				SetOrderLight := types.SetOrderLight{MasterID: MasterStruct.CurrentMasterID, ID: slaveMsg.ID, LightOn: AllRequests}
				masterSetOrderLight <- SetOrderLight
			}
			NewEvent <- MasterStruct

		case a := <-NewAction:
			//Send NewMAsterID to for all new actions (in new slave case)
			//if extra info == NewAction.extra info
			//Send NewMAsterIDch
			NewMasterIDCh <- types.NewMasterID{SlaveID: a.ID, NewMasterID: MasterStruct.CurrentMasterID}
			fmt.Println("NewAction: ")
			fmt.Println(a.Action)
			if entry, ok := MasterStruct.HRAInput.States[a.ID]; ok {
				entry.Behaviour = a.Action.Behaviour
				entry.Dirn = elevio.MotorDirToString(a.Action.Dirn)
				MasterStruct.HRAInput.States[a.ID] = entry
			}
			if MasterStruct.HRAInput.States[a.ID].Behaviour == elevator.EB_DoorOpen {
				fmt.Println("Elevator DoorOpen")
				commandDoorOpen <- types.DoorOpen{MasterID: MasterStruct.CurrentMasterID, ID: a.ID, SetDoorOpen: true}
			} else {
				fmt.Println("Master give Direction:")
				fmt.Println(a.Action.Dirn)
				masterCommandMD <- types.MasterCommand{MasterID: MasterStruct.CurrentMasterID, ID: a.ID, Motordir: elevio.MotorDirToString(a.Action.Dirn)}
			}
		}
	}
}
