package requests

import (
	"Driver-go/elevio"
)

type Action struct{
	dirn elevio.MotorDirection
	behaviour ElevatorBehaviour
}


func RequestsAbove(e Elevator) int {
	for i := e.floor + 1; i < numFloors; i++ {
		for btn := 0; btn < _numButtonTypes; btn++ {
			if e.requests[i][btn] {
				return 1
			}
		}
	}
	return 0
}

func RequestsBelow(e Elevator) int {
	for i := 0; i < e.floor; i++ {
		for btn := 0; btn < _numButtonTypes; btn++ {
			if e.requests[i][btn] {
				return 1
			}
		}
	}
	return 0
}

func RequestsHere(e Elevator) int {
	for btn := 0; btn < _numButtonTypes; btn++ {
		if e.requests[e.floor][btn] {
			return 1
		}
	}
	return 0
}

func RequestsNextAction(e Elevator) Action {
	switch e.dirn {
	case D_UP:
		if RequestsAbove(e){
			return Action{MD_Up, EB_Moving}
		}
		else if RequestsHere(e){
			return Action{MD_Down, EB_DoorOpen }
		}
		else if RequestsBelow(e){
			return Action{MD_Down, EB_Moving}
		}
		else{
			return Action{MD_Stop, EB_Idle}
		}
	}
	case 
}

func NewOrder(f)
