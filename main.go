package main

//Slave executes its own cab requests

//Fix logic in request-execution in master
//Master assigns request, divides it in to the simplest steps, saves which elevator executes the request.

//Message from slave to master -
//Need to include: New buttonevents + new floor + iterator/versionnr
//Alive message

//Message from master to slave -
//Need to include: New motor direction + open door + set lights

//Master-message
//Need to include: All relevant info + iterate/versionnr

import (
	"encoding/json"
	"fmt"
	"master/Driver-go/elevio"
	"master/elevator"
	"master/fsm"
	"master/network/broadcast"
	"master/requests"
	"os/exec"
	"runtime"
	"time"
)

type SlaveButtonEventMsg struct {
	Btn_floor int
	Btn_type  int
}

type MasterAckOrderMsg struct {
	Btn_floor int
	Btn_type  int
}

type HRAElevState struct {
	Behavior    string                 `json:"behaviour"`
	Floor       int                    `json:"floor"`
	Direction   string                 `json:"direction"`
	CabRequests [elevio.NumFloors]bool `json:"cabRequests"`
}

type HRAInput struct {
	HallRequests [][2]bool               `json:"hallRequests"`
	States       map[string]HRAElevState `json:"states"`
}

var e1 elevator.Elevator
var MasterRequests requests.AllRequests

func main() {

	//At the moment this is just to test hallRequestAssigner
	e1HRA := HRAElevState{
		Behavior:    "idle",
		Floor:       1,                                       //jalla
		Direction:   elevio.MotorDirToString(elevio.MD_Stop), //This is important
		CabRequests: [elevio.NumFloors]bool{},
	}

	//need this as long as the rest of the code isnt rewritten to e1HRA
	e1 := elevator.Elevator{
		Floor:     1, //jalla
		Dirn:      elevio.MD_Stop,
		Behaviour: elevator.EB_Idle,
	}
	if e1.Floor == -1 {
		e1 = fsm.Fsm_onInitBetweenFloors(e1)
		e1HRA.Floor = e1.Floor
	}

	fmt.Println(e1HRA)
	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "hall_request_assigner"
	case "windows":
		hraExecutable = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}

	//Using elevatorstate as input, HallRequests need to be replaced with MasterRequests
	MasterStruct := HRAInput{
		HallRequests: [][2]bool{{false, false}, {true, false}, {false, false}, {false, true}},
		States: map[string]HRAElevState{
			"one": e1HRA,
		},
	}

	input := MasterStruct

	jsonBytes, err := json.Marshal(input)
	fmt.Println("json.Marshal error: ", err)

	ret, err := exec.Command("hall_request_assigner/"+hraExecutable, "-i", string(jsonBytes)).Output()
	fmt.Println("exec.Command error: ", err)

	output := new(map[string][][2]bool)
	err = json.Unmarshal(ret, &output)
	fmt.Println("json.Unmarshal error: ", err)

	fmt.Printf("output: \n")
	for k, v := range *output {
		fmt.Printf("%6v :  %+v\n", k, v)
	}

	slaveButtonRx := make(chan SlaveButtonEventMsg)
	slaveFloorRx := make(chan int)
	masterCommandMD := make(chan int)
	masterAckOrder := make(chan MasterAckOrderMsg)
	slaveAckOrderDoneRx := make(chan bool)
	masterTurnOffOrderLightTx := make(chan int)
	slaveState := make(chan elevator.Elevator)
	commandDoorOpen := make(chan bool)
	slaveDoorOpened := make(chan bool)

	go broadcast.Receiver(16513, slaveButtonRx)
	go broadcast.Receiver(16514, slaveFloorRx)
	go broadcast.Receiver(16517, slaveAckOrderDoneRx)
	go broadcast.Receiver(16519, slaveState)
	go broadcast.Receiver(16521, slaveDoorOpened)

	go broadcast.Transmitter(16515, masterCommandMD)
	go broadcast.Transmitter(16516, masterAckOrder)
	go broadcast.Transmitter(16518, masterTurnOffOrderLightTx)
	go broadcast.Transmitter(16520, commandDoorOpen)

	doorTimer := time.NewTimer(20 * time.Second)

	fmt.Println("Dir before string:")
	fmt.Println(e1.Dirn)
	fmt.Println("Dir after string:")
	fmt.Println(elevio.MotorDirToString(e1.Dirn))

	for {
		select {
		case slaveMsg := <-slaveButtonRx: //New order from slave
			//Initial order starting here (first)
			fmt.Println("Floor")
			fmt.Println(slaveMsg.Btn_floor)
			fmt.Println("Button type")
			fmt.Println(slaveMsg.Btn_type)
			fmt.Println(" ")

			MasterRequests.Requests[slaveMsg.Btn_floor][slaveMsg.Btn_type] = true //Saving requests in master
			fmt.Println(MasterRequests)

			if e1.Behaviour == elevator.EB_Idle {
				nextAction := requests.MasterRequestsNextAction(e1, MasterRequests)
				e1.Behaviour = nextAction.Behaviour
				e1.Dirn = nextAction.Dirn
				masterCommandMD <- int(e1.Dirn)
			}

			masterAckOrder <- MasterAckOrderMsg{int(slaveMsg.Btn_floor), slaveMsg.Btn_type}

		case slaveMsg := <-slaveFloorRx: //New floor from slave
			//Next action given here, but only in same direction of elevator(?)
			e1.Floor = slaveMsg
			fmt.Println("Behaviour")
			fmt.Println(e1.Behaviour)
			fmt.Println("direction")
			fmt.Println(e1.Dirn)
			a := requests.MasterRequestsNextAction(e1, MasterRequests)
			e1.Behaviour = a.Behaviour
			e1.Dirn = a.Dirn
			if e1.Behaviour == elevator.EB_DoorOpen {
				commandDoorOpen <- true
			} else {
				masterCommandMD <- int(a.Dirn)
			}

		case slaveMsg := <-slaveDoorOpened:
			if slaveMsg {
				e1.Behaviour = elevator.EB_DoorOpen
				MasterRequests.Requests[e1.Floor][0] = false
				MasterRequests.Requests[e1.Floor][1] = false
				MasterRequests.Requests[e1.Floor][2] = false
				masterTurnOffOrderLightTx <- e1.Floor
				doorTimer.Stop()
				doorTimer.Reset(3 * time.Second)
			} else {
				a := requests.MasterRequestsNextAction(e1, MasterRequests)
				e1.Dirn = a.Dirn
				e1.Behaviour = a.Behaviour

				switch e1.Behaviour {
				case elevator.EB_DoorOpen:
					e1, MasterRequests = requests.MasterClearRequestCurrentFloor(e1, MasterRequests)
					//fsm.SetAllLights(e1)
					//elevio.SetButtonLamp(elevio.BT_HallDown, e.Floor, false)
					masterTurnOffOrderLightTx <- e1.Floor
				case elevator.EB_Moving:
					masterCommandMD <- int(e1.Dirn)
				case elevator.EB_Idle:
					masterCommandMD <- int(e1.Dirn)
				}
			}
		case <-doorTimer.C:
			commandDoorOpen <- false
			e1.Behaviour = elevator.EB_Idle
		}
	}

}

//Add check to see if order queue is empty or if new order need to be executed

//Dont tell elevator to move inside buttonevent

//include functionality from onDoorTimeout to check if elevator should continue
//moving after opening door in a floor
