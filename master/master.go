package master

import (
	"encoding/json"
	"fmt"
	"master/Driver-go/elevio"
	"master/elevator"
	"master/requests"
	"master/types"
	"os/exec"
)

//Finds next action
func MasterFindNextAction(
	NewEvent <-chan types.MasterStruct,
	NewAction chan<- types.NewAction,
	commandDoorOpen chan<- types.DoorOpen,
	masterCommandMD chan<- types.MasterCommand) {
	hraExecutable := "hall_request_assigner"
	output := new(map[string][][2]bool)
	for {
		MasterStruct := <-NewEvent
		fmt.Println("NewEventMasterStruct: ")
		fmt.Println(MasterStruct)
		HRAInput := MasterStruct.HRAInput
		input := HRAInput
		jsonBytes, _ := json.Marshal(input)

		ret, err := exec.Command("../hall_request_assigner/"+hraExecutable, "-i", string(jsonBytes)).Output()
		fmt.Println(err)
		err = json.Unmarshal(ret, &output)
		fmt.Println(MasterStruct.MySlaves)

		for _, peer := range MasterStruct.MySlaves {
			fmt.Println("MySlaves: ")
			fmt.Println(MasterStruct.MySlaves)
			ElevatorHallReqs := (*output)[peer]
			elevState := HRAInput.States[peer]
			ElevatorCabRequests := elevState.CabRequests
			var action requests.Action
			AllRequests := requests.RequestsAppendHallCab(ElevatorHallReqs, ElevatorCabRequests)
			if requests.RequestShouldStop(elevState, AllRequests) && elevState.Behaviour != "moving" {
				if requests.RequestsHere(elevState, AllRequests) {
					action = requests.Action{Dirn: elevio.StringToMotorDir(elevState.Dirn), Behaviour: elevator.EB_DoorOpen}
				} else {
					action = requests.Action{Dirn: elevio.MD_Stop, Behaviour: elevator.EB_Idle}
				}
			} else {
				if elevState.Behaviour == elevator.EB_Moving {
					action = requests.Action{Dirn: elevio.StringToMotorDir(elevState.Dirn), Behaviour: elevState.Behaviour}
				} else {
					action = requests.RequestsNextAction(elevState, AllRequests)
				}
			}
			NextAction := types.NewAction{ID: peer, Action: action}
			//Return extra info
			NewAction <- NextAction
		}
	}
}

func MergeMasterStructs(MasterStruct types.MasterStruct, ReceivedMergeStruct types.MasterStruct) types.MasterStruct {
	NewMasterStruct := MasterStruct
	MasterHallRequests := MasterStruct.HRAInput.HallRequests
	ReceivedHallRequests := ReceivedMergeStruct.HRAInput.HallRequests
	ReceivedID := ReceivedMergeStruct.CurrentMasterID
	ReceivedStates := ReceivedMergeStruct.HRAInput.States[ReceivedID]
	for i := 0; i < elevio.NumFloors; i++ {
		for j := 0; j < elevio.NumButtonTypes-1; j++ {
			NewMasterStruct.HRAInput.HallRequests[i][j] = MasterHallRequests[i][j] || ReceivedHallRequests[i][j]
		}
		if entry, ok := MasterStruct.HRAInput.States[ReceivedID]; ok {
			entry.CabRequests[i] = MasterStruct.HRAInput.States[ReceivedID].CabRequests[i] || ReceivedStates.CabRequests[i]
			entry.Behaviour = ReceivedStates.Behaviour
			entry.Dirn = ReceivedStates.Dirn
			entry.Floor = ReceivedStates.Floor
			NewMasterStruct.HRAInput.States[ReceivedID] = entry
		}
	}
	return NewMasterStruct
}
