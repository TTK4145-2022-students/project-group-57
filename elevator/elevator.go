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
	Behaviour   ElevatorBehaviour
	Floor       int
	Dirn        elevio.MotorDirection
	CabRequests [elevio.NumFloors]bool
}

type NewElevator struct {
	Behaviour   ElevatorBehaviour      `json:"behaviour"`
	Floor       int                    `json:"floor"`
	Dirn        elevio.MotorDirection  `json:"direction"`
	CabRequests [elevio.NumFloors]bool `json:"cabRequests"`
}

type StateStruct struct {
	ID                 string
	LocalElevatorState NewElevator
}

func ElevatorUninitialized() Elevator {
	uninitElevator := Elevator{
		Floor:     -1,
		Dirn:      elevio.MD_Stop,
		Behaviour: EB_Idle,
	}
	return uninitElevator
}
