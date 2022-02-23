package requests

import (
	"Driver-go/elevio"
	"elevator"
)

type Action struct {
	dirn      elevio.MotorDirection
	behaviour ElevatorBehaviour
}

func RequestsAbove(e elevator.Elevator) int {
	for i := e.floor + 1; i < elevio._numFloors; i++ {
		for btn := 0; btn < _elevio._numButtonTypes; btn++ {
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
	case MD_Up:
		if RequestsAbove(e) {
			return Action{MD_Up, EB_Moving}
		} else if RequestsHere(e) {
			return Action{MD_Down, EB_DoorOpen}
		} else if RequestsBelow(e) {
			return Action{MD_Down, EB_Moving}
		} else {
			return Action{MD_Stop, EB_Idle}
		}
	case MD_Down:
		if RequestsBelow(e) {
			return Action{MD_Down, EB_Moving}
		} else if RequestsHere(e) {
			return Action{MD_Up, EB_DoorOpen}
		} else if RequestsAbove(e) {
			return Action{MD_Up, EB_Moving}
		} else {
			return Action{MD_Stop, EB_Idle}
		}
	case MD_Stop:
		if RequestsHere(e) {
			return Action{MD_Stop, EB_DoorOpen}
		} else if RequestsAbove(e) {
			return Action{MD_Up, EB_Moving}
		} else if RequestsBelow(e) {
			return Action{MD_Down, EB_Moving}
		} else {
			return Action{MD_Stop, EB_Idle}
		}
	default:
		return Action{MD_Stop, EB_Idle}
	}
}

func RequestShouldStop(e Elevator) int {
	switch e.dirn {
	case MD_Down:
		return e.requests[e.floor][BT_HallDown] || e.requests[e.floor][BT_Cab] || !RequestsBelow(e)
	case MD_Up:
		return e.requests[e.floor][BT_HallUp] || e.requests[e.floor][BT_Cab] || !Requestsabove(e)
	case MD_Stop:
		return 1
	default:
		return 1
	}
}

func ClearRequestImmediately(e Elevator, btnFloor int, btnType ButtonType) int {
	if e.floor == btnFloor {
		if e.dirn == MD_Up && btnType == BT_HallUp {
			return 1
		} else if e.dirn == MD_Down && btnType == BT_HallDown {
			return 1
		} else if e.dirn == MD_Stop {
			return 1
		} else if btnType == BT_Cab {
			return 1
		}
	} else {
		return 0
	}
}

//Clears all requests in the floor when the elevator stops
//This might have to be changed, assumes that everyone enters the elevator in the floor, regardless of direction
func ClearRequestCurrentFloor(e Elevator) {
	e.requests[e.floor][BT_Cab] = 0
	e.requests[e.floor][BT_HallUp] = 0
	e.requests[e.floor][BT_HallDown] = 0
}
