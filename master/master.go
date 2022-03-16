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

func MasterGiveCommands(NewEvent <-chan types.HRAInput, NewAction chan<- requests.Action, commandDoorOpen chan<- types.DoorOpen, masterCommandMD chan<- types.MasterCommand) {
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

		ElevatorHallReqs := (*output)["one"] //iterate
		elevState := MasterStruct.States["one"]

		a := requests.RequestsNextAction(elevState, ElevatorHallReqs)

		NewAction <- a

		if MasterStruct.States["one"].Behaviour == elevator.EB_DoorOpen {
			commandDoorOpen <- types.DoorOpen{ID: "one", SetDoorOpen: true}
		} else {
			masterCommandMD <- types.MasterCommand{ID: "one", Motordir: elevio.MotorDirToString(a.Dirn)}
		}
	}
}
