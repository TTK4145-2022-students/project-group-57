package orders

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

func OrderAbove(e Elevator) int {
	for i := e.floor + 1; i < numFloors; i++ {
		for btn := 0; btn < _numButtonTypes; btn++ {
			if e.requests[i][btn] {
				return 1
			}
		}
	}
	return 0
}

func OrderBelow(e Elevator) int {
	for i := 0; i < e.floor; i++ {
		for btn := 0; btn < _numButtonTypes; btn++ {
			if e.requests[i][btn] {
				return 1
			}
		}
	}
	return 0
}

func OrderHere(e Elevator) int {
	for btn := 0; btn < _numButtonTypes; btn++ {
		if e.requests[e.floor][btn] {
			return 1
		}
	}
	return 0
}

func NewOrder(f)
