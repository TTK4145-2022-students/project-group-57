package elevator

import (
	"master/Driver-go/elevio"
)

type ElevatorBehaviour string

const (
	EB_Idle     ElevatorBehaviour = "idle"
	EB_Moving   ElevatorBehaviour = "moving"
	EB_DoorOpen ElevatorBehaviour = "doorOpen"
)

type Elevator struct {
	Behaviour   ElevatorBehaviour      `json:"behaviour"`
	Floor       int                    `json:"floor"`
	Dirn        string                 `json:"direction"`
	CabRequests [elevio.NumFloors]bool `json:"cabRequests"`
}

type SingleElevator struct {
	Floor     int
	Dirn      string
	Requests  [elevio.NumFloors][elevio.NumButtonTypes]bool
	Behaviour ElevatorBehaviour
}

func ElevatorUninitialized() Elevator {
	uninitElevator := Elevator{
		Behaviour: EB_Idle,
		Floor:     -1,
		Dirn:      elevio.MotorDirToString(elevio.MD_Stop),
	}
	return uninitElevator
}
