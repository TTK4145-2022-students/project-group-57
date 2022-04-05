package main

//Problems:
//HallUp and HallDown in current floor, Cab in diff floor after HallUp is executed, but before HallDown is executed
//---- Elev doesn't execute cab before HallDown
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

//Single elevator can have info from multiple elevs in its masterstruct
//two cases

func main() {

	slaveButtonRx := make(chan types.SlaveButtonEventMsg)
	slaveFloorRx := make(chan types.SlaveFloor, 5)
	masterCommandMD := make(chan types.MasterCommand)
	masterSetOrderLight := make(chan types.SetOrderLight, 3)
	commandDoorOpen := make(chan types.DoorOpen, 3)
	slaveDoorOpened := make(chan types.DoorOpen, 3)
	NewEvent := make(chan types.MasterStruct, 5)
	NewAction := make(chan types.NewAction, 5)
	PeerUpdateCh := make(chan peers.PeerUpdate)
	MasterMsg := make(chan types.MasterStruct, 3)
	MasterInitStruct := make(chan types.MasterStruct)
	MasterMergeSend := make(chan types.MasterStruct, 3)
	MasterMergeReceive := make(chan types.MasterStruct, 3)

	NewMasterIDCh := make(chan types.NewMasterID)
	UnableToMoveCh := make(chan types.UnableToMove, 3)

	go broadcast.Receiver(16513, slaveButtonRx)
	go broadcast.Receiver(16514, slaveFloorRx)
	go broadcast.Receiver(16521, slaveDoorOpened)
	go broadcast.Receiver(16527, MasterInitStruct)
	go broadcast.Transmitter(16515, masterCommandMD)
	go broadcast.Transmitter(16518, masterSetOrderLight)
	go broadcast.Transmitter(16520, commandDoorOpen)
	go master.MasterFindNextAction(NewEvent, NewAction)
	go broadcast.TransmitMasterMsg(16523, MasterMsg)
	go broadcast.Transmitter(16524, NewMasterIDCh)
	go broadcast.Transmitter(16585, MasterMergeSend)
	go broadcast.Receiver(16585, MasterMergeReceive)
	go broadcast.Receiver(16528, UnableToMoveCh)

	go peers.Receiver(16529, PeerUpdateCh)

	const interval = 500 * time.Millisecond
	var NewPeerList peers.PeerUpdate

	initArg := os.Args[1]
	isolatedArg := os.Args[2]

	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println(initArg)

	var MasterStruct types.MasterStruct
	MasterStruct.ElevStates = map[string]elevator.Elev{}
	MasterStruct.Initialized = false

	if initArg == "init" {
		//start timer
		//Wait for initstruct from slave
		//if timer out -> ded
		//send struct to mastermergesend
		//ded

		SlaveID := os.Args[2]
		CurrentFloor := os.Args[3]
		fmt.Println(SlaveID)
		fmt.Println(CurrentFloor)
		fmt.Println("init master")
		MasterStruct = types.MasterStruct{
			CurrentMasterID: SlaveID,
			Isolated:        true,  //keep one of these?
			Initialized:     false, //keep one of these?
			PeerList:        peers.PeerUpdate{},
			HallRequests:    [][2]bool{{false, false}, {false, false}, {false, false}, {false, false}},
			MySlaves:        []string{SlaveID},
			ElevStates:      map[string]elevator.Elev{},
		}
		MasterStruct.ElevStates[SlaveID] = fsm.UnInitializedElev()
		fmt.Println(MasterStruct)
		if entry, ok := MasterStruct.ElevStates[SlaveID]; ok {
			fmt.Print("entry")
			entry.Floor, _ = strconv.Atoi(CurrentFloor)
			MasterStruct.ElevStates[SlaveID] = entry
			fmt.Println(MasterStruct)
		}
		//Sends to already existing master
		//Send for a longer time?
		//
		for i := 0; i < 5; i++ {
			MasterMergeSend <- MasterStruct //Init false
			time.Sleep(300 * time.Millisecond)
		}
		fmt.Println("Time to kill")
		os.Exit(99)
	}

	MasterStruct = <-MasterInitStruct
	MasterStruct.Initialized = true
	IsolatedMasterStruct := MasterStruct
	if isolatedArg == "isolated" {
		go func() {
			for i := 0; i < 5; i++ {
				MasterMergeSend <- IsolatedMasterStruct
				time.Sleep(300 * time.Millisecond)
			}
		}()
	}

	for {
		select {
		case a := <-UnableToMoveCh:
			fmt.Println("UnableToMove received, value: ")
			fmt.Println(a.UnableToMove)
			if a.UnableToMove {
				MasterStruct.MySlaves = master.DeleteLostPeer(MasterStruct.MySlaves, a.ID)
				fmt.Println()
				fmt.Println()
				fmt.Println()
				fmt.Println()
				fmt.Println("***********************************Deleted Slave*****************************")
				fmt.Println(a.ID)
				fmt.Println()
				fmt.Println()
				fmt.Println()
			} else {
				MasterStruct.MySlaves = master.AppendNoDuplicates(MasterStruct.MySlaves, a.ID)
			}
			NewEvent <- MasterStruct

		case <-time.After(interval):
			MasterMsg <- MasterStruct

		case ReceivedMergeStruct := <-MasterMergeReceive:
			fmt.Println("Case ReceivedMasterStruct")
			fmt.Println("Received: ")
			fmt.Println(ReceivedMergeStruct)
			fmt.Println("Current masterstruct: ")
			fmt.Println(MasterStruct)

			var NextInLine string
			if len(ReceivedMergeStruct.PeerList.Peers) < 2 {
				NextInLine = MasterStruct.CurrentMasterID
			} else {
				NextInLine = ReceivedMergeStruct.PeerList.Peers[0]
			}
			if master.ShouldStayMaster(MasterStruct.CurrentMasterID, NextInLine, MasterStruct.Isolated, ReceivedMergeStruct.Isolated) {
				MasterStruct = master.MergeMasterStructs(MasterStruct, ReceivedMergeStruct)
				fmt.Println("Merged struct: ")
				fmt.Println(MasterStruct)
				HallRequests := MasterStruct.HallRequests
				for k := range MasterStruct.ElevStates {
					CabRequests := MasterStruct.ElevStates[k].CabRequests
					AllRequests := requests.RequestsAppendHallCab(HallRequests, CabRequests)
					SetOrderLight := types.SetOrderLight{MasterID: MasterStruct.CurrentMasterID, ID: k, LightOn: AllRequests}
					masterSetOrderLight <- SetOrderLight
				}
			} else {
				for i := 0; i < 5; i++ {
					MasterMergeSend <- MasterStruct
					time.Sleep(300 * time.Millisecond)
				}
				fmt.Println("Time to kill")
				os.Exit(99)
			}

			NewEvent <- MasterStruct

		case NewPeerList = <-PeerUpdateCh: //Use only for deleting, not adding new
			//Periodically add slaves to myslaves from peerlist.peers

			fmt.Println("Peerlist")
			fmt.Println(NewPeerList)
			MasterStruct.PeerList = NewPeerList
			if len(NewPeerList.Lost) != 0 {
				for k := range NewPeerList.Lost {
					MasterStruct.MySlaves = master.DeleteLostPeer(MasterStruct.MySlaves, NewPeerList.Lost[k])
				}
				fmt.Println(MasterStruct.MySlaves)
				NewEvent <- MasterStruct
			}

		case slaveMsg := <-slaveButtonRx:
			if slaveMsg.Btn_type == 2 {
				if entry, ok := MasterStruct.ElevStates[slaveMsg.ID]; ok {
					entry.CabRequests[slaveMsg.Btn_floor] = true
					MasterStruct.ElevStates[slaveMsg.ID] = entry
				}
			} else {
				MasterStruct.HallRequests[slaveMsg.Btn_floor][slaveMsg.Btn_type] = true
			}
			AllRequests := requests.RequestsAppendHallCab(MasterStruct.HallRequests, MasterStruct.ElevStates[slaveMsg.ID].CabRequests)
			SetOrderLight := types.SetOrderLight{MasterID: MasterStruct.CurrentMasterID, ID: slaveMsg.ID, LightOn: AllRequests}
			NewEvent <- MasterStruct
			masterSetOrderLight <- SetOrderLight

		case slaveMsg := <-slaveFloorRx:
			elevatorID := slaveMsg.ID
			elevatorFloor := slaveMsg.NewFloor
			if entry, ok := MasterStruct.ElevStates[elevatorID]; ok {
				entry.Floor = elevatorFloor
				entry.Behaviour = elevator.EB_Idle
				MasterStruct.ElevStates[elevatorID] = entry
			}
			NewEvent <- MasterStruct

		case slaveMsg := <-slaveDoorOpened:
			fmt.Println("ID: ")
			fmt.Println(slaveMsg.ID)
			fmt.Println("SetDoorOpen: ")
			fmt.Println(slaveMsg.SetDoorOpen)
			elevState := MasterStruct.ElevStates[slaveMsg.ID]
			if slaveMsg.SetDoorOpen {
				if entry, ok := MasterStruct.ElevStates[slaveMsg.ID]; ok {
					entry.Behaviour = elevator.EB_DoorOpen
					entry.CabRequests[elevState.Floor] = false
					MasterStruct.ElevStates[slaveMsg.ID] = entry
				}
				elevState = MasterStruct.ElevStates[slaveMsg.ID]
				ClearHallReqs := requests.ShouldClearHallRequest(elevState, MasterStruct.HallRequests)
				MasterStruct.HallRequests[elevState.Floor][elevio.BT_HallUp] = ClearHallReqs[elevio.BT_HallUp]
				MasterStruct.HallRequests[elevState.Floor][elevio.BT_HallDown] = ClearHallReqs[elevio.BT_HallDown]
				AllRequests := requests.RequestsAppendHallCab(MasterStruct.HallRequests, MasterStruct.ElevStates[slaveMsg.ID].CabRequests)
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
			if entry, ok := MasterStruct.ElevStates[a.ID]; ok {
				entry.Behaviour = a.Action.Behaviour
				entry.Dirn = elevio.MotorDirToString(a.Action.Dirn)
				MasterStruct.ElevStates[a.ID] = entry
			}
			if MasterStruct.ElevStates[a.ID].Behaviour == elevator.EB_DoorOpen {
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
