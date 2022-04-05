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
		for _, ID := range MasterStruct.ActiveSlaves {
			HRAInput.States[ID] = MasterStruct.ElevStates[ID]
		}
		jsonBytes, _ := json.Marshal(HRAInput)
		ret, err := exec.Command("../hall_request_assigner/"+HRAExecutable, "-i", string(jsonBytes)).Output()
		fmt.Println(err)
		err = json.Unmarshal(ret, &HRAOutput)

		fmt.Printf("output: \n")
		for k, v := range *HRAOutput {
			fmt.Printf("%6v :  %+v\n", k, v)
		}
		fmt.Println(MasterStruct.ActiveSlaves)
		for _, peer := range MasterStruct.PeerList.Peers {
			var NextAction types.NewAction
			isActive := false
			for _, slave := range MasterStruct.ActiveSlaves {
				if peer == slave {
					isActive = true
				}
			}
			if isActive {
				ElevAssignedHallReqs := (*HRAOutput)[peer]
				elevState := HRAInput.States[peer]
				ElevCabRequests := elevState.CabRequests
				var action requests.Action
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
				NextAction = types.NewAction{ID: peer, Action: action}
			} else {
				previousDirn := elevio.StringToMotorDir(MasterStruct.ElevStates[peer].Dirn)
				if previousDirn == elevio.MD_Stop {
					if MasterStruct.ElevStates[peer].Floor == 0 {
						previousDirn = elevio.MD_Up
					} else {
						previousDirn = elevio.MD_Down
					}
				}
				if MasterStruct.ElevStates[peer].Floor == 0 && previousDirn == elevio.MD_Down {
					previousDirn = elevio.MD_Up
				}
				fmt.Println("Current peer: ")
				fmt.Println(peer)
				fmt.Println("NextAction: ")
				fmt.Println(previousDirn)
				action := requests.Action{Dirn: previousDirn, Behaviour: elevator.EB_Moving}
				NextAction = types.NewAction{ID: peer, Action: action}
			}
			NewAction <- NextAction
		}
		//Return extra info
	}
}

func MergeMasterStructs(MasterStruct types.MasterStruct, ReceivedMergeStruct types.MasterStruct) types.MasterStruct {
	//Check if Received can have multiple elevstate
	MasterHallRequests := MasterStruct.HallRequests
	ReceivedHallRequests := ReceivedMergeStruct.HallRequests
	ReceivedID := ReceivedMergeStruct.CurrentMasterID
	ReceivedState := ReceivedMergeStruct.ElevStates[ReceivedID]
	MasterStruct.ActiveSlaves = AppendNoDuplicates(MasterStruct.ActiveSlaves, ReceivedID)
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

func AppendNoDuplicates(ActiveSlaves []string, Peer string) []string {
	duplicate := false
	for _, slave := range ActiveSlaves {
		if Peer == slave {
			duplicate = true
			break
		}
	}
	if !duplicate {
		ActiveSlaves = append(ActiveSlaves, Peer)
	}
	return ActiveSlaves
}

func DeleteLostPeer(ActiveSlaves []string, LostPeers string) []string {
	var UpdatedActiveSlaves []string
	for j := range ActiveSlaves {
		if LostPeers != ActiveSlaves[j] {
			UpdatedActiveSlaves = append(UpdatedActiveSlaves, ActiveSlaves[j])
		}
	}
	return UpdatedActiveSlaves
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
