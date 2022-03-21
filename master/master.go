package master

import (
	"encoding/json"
	"fmt"
	"master/Driver-go/elevio"
	"master/elevator"
	"master/network/peers"
	"master/requests"
	"master/types"
	"os/exec"
)

//Finds next action
func MasterGiveCommands(NewEvent <-chan types.HRAInput, NewAction chan<- types.NewAction, commandDoorOpen chan<- types.DoorOpen, masterCommandMD chan<- types.MasterCommand, PeerList peers.PeerUpdate) {
	hraExecutable := "hall_request_assigner"
	output := new(map[string][][2]bool)
	for {
		MasterStruct := <-NewEvent

		input := MasterStruct
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

		for _, peer := range PeerList.Peers {
			ElevatorHallReqs := (*output)[peer]
			elevState := MasterStruct.States[peer]
			ElevatorCabRequests := elevState.CabRequests
			var action requests.Action
			AllRequests := requests.RequestsAppendHallCab(ElevatorHallReqs, ElevatorCabRequests)
			if elevState.Behaviour == elevator.EB_DoorOpen || elevState.Behaviour == elevator.EB_Idle {
				action = requests.RequestsNextAction(elevState, AllRequests)
			} else {
				if requests.RequestShouldStop(elevState, AllRequests) {
					action = requests.Action{Dirn: elevio.StringToMotorDir(elevState.Dirn), Behaviour: elevator.EB_DoorOpen}
				} else {
					action = requests.Action{Dirn: elevio.StringToMotorDir(elevState.Dirn), Behaviour: elevState.Behaviour}
				}
			}
			NextAction := types.NewAction{ID: peer, Action: action}
			if !(elevState.Behaviour == elevator.EB_DoorOpen && NextAction.Action.Behaviour == elevator.EB_DoorOpen) {
				NewAction <- NextAction
			}
		}
	}
}
