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
func MasterFindNextAction(NewEvent <-chan types.HRAInput, NewAction chan<- types.NewAction, commandDoorOpen chan<- types.DoorOpen, masterCommandMD chan<- types.MasterCommand, PeerList peers.PeerUpdate) {
	hraExecutable := "hall_request_assigner"
	output := new(map[string][][2]bool)
	for {
		MasterStruct := <-NewEvent

		elevBehav1 := MasterStruct.States["one"].Behaviour

		elevDirn1 := MasterStruct.States["one"].Dirn

		elevFloor1 := MasterStruct.States["one"].Floor

		fmt.Println("")
		fmt.Println("")
		fmt.Println("")
		fmt.Println("HRAOne")
		fmt.Println(elevBehav1)
		fmt.Println(elevDirn1)
		fmt.Println(elevFloor1)
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
			fmt.Println(elevState.Behaviour)
			ElevatorCabRequests := elevState.CabRequests
			var action requests.Action
			AllRequests := requests.RequestsAppendHallCab(ElevatorHallReqs, ElevatorCabRequests)

			if requests.RequestShouldStop(elevState, AllRequests) && elevState.Behaviour != "moving" {
				fmt.Println(peer)
				fmt.Println("Should stop")
				fmt.Println(elevState.Floor)
				if requests.RequestsHere(elevState, AllRequests) {
					action = requests.Action{Dirn: elevio.StringToMotorDir(elevState.Dirn), Behaviour: elevator.EB_DoorOpen}
				} else {
					action = requests.Action{Dirn: elevio.MD_Stop, Behaviour: elevator.EB_Idle}
				}
			} else {
				if elevState.Behaviour == elevator.EB_Moving {
					fmt.Println(peer)
					fmt.Println("Keeping dir")
					action = requests.Action{Dirn: elevio.StringToMotorDir(elevState.Dirn), Behaviour: elevState.Behaviour}
				} else {
					action = requests.RequestsNextAction(elevState, AllRequests)
				}

			}

			NextAction := types.NewAction{ID: peer, Action: action}
			NewAction <- NextAction
		}
	}
}
