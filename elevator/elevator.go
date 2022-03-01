package elevator

import (
	"master/Driver-go/elevio"
)

type ElevatorBehaviour int64

const (
	EB_Idle ElevatorBehaviour = 0
	EB_Moving  ElevatorBehaviour = 1
	EB_DoorOpen ElevatorBehaviour = -1
)

type Elevator struct {
	Floor     int
	Dirn      elevio.MotorDirection
	Requests  [elevio._numFloors][elevio.ButtonType]int
	Behaviour ElevatorBehaviour
}

func ElevatorUninitialized() Elevator {
	uninitElevator := Elevator{
		Floor: -1,
		Dirn: elevio.MD_Stop,
		Requests: [nil][nil],
		Behaviour: EB_Idle,
	}
	return uninitElevator
}