package elevator

import (
	"Driver-go/elevio"
)

type ElevatorBehaviour int64

const (
	Idle ElevatorBehaviour = 0
	Up   ElevatorBehaviour = 1
	Down ElevatorBehaviour = -1
)

type Elevator struct {
	floor     int
	dirn      elevio.MotorDirection
	requests  [elevio._numFloors][elevio.ButtonType]int
	behaviour ElevatorBehaviour
}

func ElevatorUninitialized() Elevator {
	uninitElevator := Elevator{
		floor: -1,
		dirn: elevio.MotorDirection: MD_Stop,
		requests: [],
		behaviour: Idle,
	}
}
