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
	NewAction chan<- types.NewAction) {
	HRAExecutable := "hall_request_assigner"
	HRAOutput := new(map[string][][2]bool)
	for {
		MasterStruct := <-NewEvent
		HRAInput := types.HRAInput{
			HallRequests: MasterStruct.HallRequests,
			States:       map[string]elevator.Elev{},
		}
		//Iterate Myslaves
		//Save input = ElevStates[myslaveID]

		//////////////////////////////////////////
		for _, ID := range MasterStruct.MySlaves.Active {
			HRAInput.States[ID] = MasterStruct.ElevStates[ID]
		}
		//////////////////////////////////////////////
		jsonBytes, _ := json.Marshal(HRAInput)
		ret, _ := exec.Command("../hall_request_assigner/"+HRAExecutable, "-i", string(jsonBytes)).Output()
		json.Unmarshal(ret, &HRAOutput)

		fmt.Printf("output: \n")
		for k, v := range *HRAOutput {
			fmt.Printf("%6v :  %+v\n", k, v)
		}
		fmt.Println(MasterStruct.MySlaves.Active)
		var NextAction types.NewAction
		var action requests.Action
		for _, slave := range MasterStruct.MySlaves.Active {
			ElevAssignedHallReqs := (*HRAOutput)[slave]
			elevState := HRAInput.States[slave]
			ElevCabRequests := elevState.CabRequests
			AllRequests := requests.RequestsAppendHallCab(ElevAssignedHallReqs, ElevCabRequests)
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
			NextAction = types.NewAction{ID: slave, Action: action}
			NewAction <- NextAction
		}
		for _, slave := range MasterStruct.MySlaves.Immobile {
			NextAction = types.NewAction{ID: slave, Action: action}
			previousDirn := elevio.StringToMotorDir(MasterStruct.ElevStates[slave].Dirn)
			if previousDirn == elevio.MD_Stop {
				if MasterStruct.ElevStates[slave].Floor == 0 {
					previousDirn = elevio.MD_Up
				} else {
					previousDirn = elevio.MD_Down
				}
			}
			if MasterStruct.ElevStates[slave].Floor == 0 && previousDirn == elevio.MD_Down {
				previousDirn = elevio.MD_Up
			}
			fmt.Println("Current slave: ")
			fmt.Println(slave)
			fmt.Println("NextAction: ")
			fmt.Println(previousDirn)
			action := requests.Action{Dirn: previousDirn, Behaviour: elevator.EB_Moving}
			NextAction = types.NewAction{ID: slave, Action: action}
			NewAction <- NextAction
		}
	}
}

func MergeMasterStructs(MasterStruct types.MasterStruct, ReceivedMergeStruct types.MasterStruct) types.MasterStruct {
	MasterHallRequests := MasterStruct.HallRequests
	ReceivedHallRequests := ReceivedMergeStruct.HallRequests
	ReceivedID := ReceivedMergeStruct.CurrentMasterID
	ReceivedState := ReceivedMergeStruct.ElevStates[ReceivedID]
	MasterStruct.MySlaves.Active = AppendNoDuplicates(MasterStruct.MySlaves.Active, ReceivedID)
	MasterStruct.Isolated = false
	if entry, ok := MasterStruct.ElevStates[ReceivedID]; ok {
		for i := 0; i < elevio.NumFloors; i++ {
			entry.CabRequests[i] = MasterStruct.ElevStates[ReceivedID].CabRequests[i] || ReceivedState.CabRequests[i]
		}
		entry.Behaviour = ReceivedState.Behaviour
		entry.Dirn = ReceivedState.Dirn
		entry.Floor = ReceivedState.Floor
		MasterStruct.ElevStates[ReceivedID] = entry
	} else {
		MasterStruct.ElevStates[ReceivedID] = ReceivedState
	}
	for i := 0; i < elevio.NumFloors; i++ {
		for j := 0; j < elevio.NumButtonTypes-1; j++ {
			MasterStruct.HallRequests[i][j] = MasterHallRequests[i][j] || ReceivedHallRequests[i][j]
		}
	}
	return MasterStruct
}

func AppendNoDuplicates(slice []string, element string) []string {
	duplicate := false
	for _, val := range slice {
		if element == val {
			duplicate = true
			break
		}
	}
	if !duplicate {
		slice = append(slice, element)
	}
	return slice
}

func DeleteElementFromSlice(slice []string, element string) []string {
	var result []string
	for j := range slice {
		if element != slice[j] {
			result = append(result, slice[j])
		}
	}
	return result
}

func ShouldStayMaster(CurrentMaster string, NextInLine string, MasterIsolated bool, ReceivedIsolated bool) bool {
	if (MasterIsolated && ReceivedIsolated) || (!MasterIsolated && !ReceivedIsolated) {
		if NextInLine == CurrentMaster {
			return true
		} else {
			return false
		}
	} else if !MasterIsolated && ReceivedIsolated {
		return true
	} else {
		return false
	}
}
