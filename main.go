package main

//Problems:
//Concurrent map read and write

//Need to check when smoothing the code:
//Consistent use of numFloors (not the number), same with numbuttons?
//Check for unneccesary types
//Rename channels, functions, variables, erthing
//Uppercase lowercase

//Fixes
//two masters coexist -- FIXED
//Elevator receiving doorOpen in floor, but elevator has powerLoss, need to remove from activeSlaves
//PacketLoss and disconnects
//Fix SingleElevator?
//MyIP instead of MyID
//Possible processPair with slave?
//Make function masterUninitMasterStructsudo 16
//SwitchCase on masterArgs?
//Two elevators come from isolated
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
	MasterInitStruct := make(chan types.MasterStruct, 2)
	MasterMergeSend := make(chan types.MasterStruct, 3)
	MasterMergeReceive := make(chan types.MasterStruct, 3)

	NewMasterIDCh := make(chan types.NewMasterID)
	AbleToMoveCh := make(chan types.AbleToMove, 3)

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
	go broadcast.Receiver(16528, AbleToMoveCh)

	go peers.Receiver(16529, PeerUpdateCh)

	const interval = 100 * time.Millisecond
	PeriodicNewEventIterator := 0
	var NewPeerList peers.PeerUpdate

	initArg := os.Args[1]
	SlaveID := os.Args[2]
	isolatedArg := os.Args[3]

	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println(initArg)
	fmt.Println(isolatedArg)

	var MasterStruct types.MasterStruct
	MasterStruct.ElevStates = map[string]elevator.Elev{}
	MasterStruct.Initialized = false

	if initArg == "init" {
		CurrentFloor := os.Args[4]
		MySlaves := types.MySlaves{Active: []string{SlaveID}}
		fmt.Println(SlaveID)
		fmt.Println(CurrentFloor)
		fmt.Println("init master")
		MasterStruct = types.MasterStruct{
			CurrentMasterID: SlaveID,
			ProcessID:       os.Getpid(),
			Isolated:        true,  //keep one of these?
			Initialized:     false, //keep one of these?
			PeerList:        peers.PeerUpdate{},
			HallRequests:    [][2]bool{{false, false}, {false, false}, {false, false}, {false, false}},
			MySlaves:        MySlaves,
			ElevStates:      map[string]elevator.Elev{},
		}
		MasterStruct.ElevStates[SlaveID] = fsm.UnInitializedElev()
		if entry, ok := MasterStruct.ElevStates[SlaveID]; ok {
			fmt.Print("entry")
			entry.Floor, _ = strconv.Atoi(CurrentFloor)
			MasterStruct.ElevStates[SlaveID] = entry
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

	fmt.Println("Waiting for initstruct")
	//Check for own initstruct
	for MasterStruct.CurrentMasterID != SlaveID {
		MasterStruct = <-MasterInitStruct
		MasterStruct.ProcessID = os.Getpid()
	}

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
		case a := <-AbleToMoveCh:
			fmt.Println("UnableToMove received, value: ")
			fmt.Println(a.AbleToMove)
			fmt.Println(MasterStruct.MySlaves.Active)
			fmt.Println(MasterStruct.MySlaves.Immobile)
			if a.AbleToMove {
				MasterStruct.MySlaves.Active = master.AppendNoDuplicates(MasterStruct.MySlaves.Active, a.ID)
				MasterStruct.MySlaves.Immobile = master.DeleteElementFromSlice(MasterStruct.MySlaves.Immobile, a.ID)
			} else {
				MasterStruct.MySlaves.Active = master.DeleteElementFromSlice(MasterStruct.MySlaves.Active, a.ID)
				MasterStruct.MySlaves.Immobile = master.AppendNoDuplicates(MasterStruct.MySlaves.Immobile, a.ID)
				fmt.Println()
				fmt.Println()
				fmt.Println()
				fmt.Println()
				fmt.Println("***********************************Deleted Slave*****************************")
				fmt.Println(a.ID)
				fmt.Println()
				fmt.Println()
				fmt.Println()
			}
			fmt.Println(MasterStruct.MySlaves.Active)
			fmt.Println(MasterStruct.MySlaves.Immobile)
			NewEvent <- MasterStruct

		case <-time.After(interval):
			//interval = 100ms
			MasterMsg <- MasterStruct
			if PeriodicNewEventIterator == 10 {
				PeriodicNewEventIterator = 0
				fmt.Println("ActiveSlaves: ")
				fmt.Println(MasterStruct.MySlaves.Active)
				MasterMergeSend <- MasterStruct
				NewEvent <- MasterStruct
			} else {
				PeriodicNewEventIterator++
			}
		case ReceivedMergeStruct := <-MasterMergeReceive:

			fmt.Println("Case ReceivedMasterStruct")
			fmt.Println("Received: ")
			fmt.Println(ReceivedMergeStruct)
			fmt.Println("Current masterstruct: ")
			fmt.Println(MasterStruct)

			if ReceivedMergeStruct.CurrentMasterID == MasterStruct.CurrentMasterID {
				if ReceivedMergeStruct.ProcessID != MasterStruct.ProcessID {
					fmt.Println("Time to kill, I am myself")
					os.Exit(99)
				}
			} else {
				var NextInLine string
				if len(ReceivedMergeStruct.PeerList.Peers) < 2 {
					NextInLine = MasterStruct.CurrentMasterID
				} else {
					NextInLine = ReceivedMergeStruct.PeerList.Peers[0]
				}
				if master.ShouldStayMaster(MasterStruct.CurrentMasterID, NextInLine, MasterStruct.Isolated, ReceivedMergeStruct.Isolated) {
					MasterStruct = master.MergeMasterStructs(MasterStruct, ReceivedMergeStruct)
					fmt.Println("merged myslaves")
					fmt.Println(MasterStruct.MySlaves.Active)
					fmt.Println(MasterStruct.MySlaves.Immobile)
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
			}

			NewEvent <- MasterStruct

		case NewPeerList = <-PeerUpdateCh: //Use only for deleting, not adding new
			//Periodically add slaves to myslaves from peerlist.peers
			fmt.Println("Peerlist")
			fmt.Println(NewPeerList)
			MasterStruct.PeerList = NewPeerList
			if len(NewPeerList.Lost) != 0 {
				for k := range NewPeerList.Lost {
					MasterStruct.MySlaves.Active = master.DeleteElementFromSlice(MasterStruct.MySlaves.Active, NewPeerList.Lost[k])
					MasterStruct.MySlaves.Immobile = master.DeleteElementFromSlice(MasterStruct.MySlaves.Immobile, NewPeerList.Lost[k])
				}
				NewEvent <- MasterStruct
			}
			fmt.Println("updated a peer")
			fmt.Println(MasterStruct.MySlaves.Active)
			fmt.Println(MasterStruct.MySlaves.Immobile)

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

			NewMasterIDCh <- types.NewMasterID{SlaveID: a.ID, NewMasterID: MasterStruct.CurrentMasterID}

			fmt.Println("ID")
			fmt.Println(a.ID)
			fmt.Println("NewAction: ")
			fmt.Println(a.Action)
			if entry, ok := MasterStruct.ElevStates[a.ID]; ok {
				entry.Behaviour = a.Action.Behaviour
				entry.Dirn = elevio.MotorDirToString(a.Action.Dirn)
				MasterStruct.ElevStates[a.ID] = entry
			}
			if MasterStruct.ElevStates[a.ID].Behaviour == elevator.EB_DoorOpen {
				commandDoorOpen <- types.DoorOpen{MasterID: MasterStruct.CurrentMasterID, ID: a.ID, SetDoorOpen: true}
			} else {
				masterCommandMD <- types.MasterCommand{MasterID: MasterStruct.CurrentMasterID, ID: a.ID, Motordir: elevio.MotorDirToString(a.Action.Dirn)}
			}
		}
	}
}
