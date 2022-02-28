package requests

import (
	"Driver-go/elevio"
	"elevator"
)

type Action struct {
	dirn      elevio.MotorDirection
	behaviour elevator.ElevatorBehaviour
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

func RequestsBelow(e elevator.Elevator) int {
	for i := 0; i < e.floor; i++ {
		for btn := 0; btn < _numButtonTypes; btn++ {
			if e.requests[i][btn] {
				return 1
			}
		}
	}
	return 0
}

func RequestsHere(e elevator.Elevator) int {
	for btn := 0; btn < _numButtonTypes; btn++ {
		if e.requests[e.floor][btn] {
			return 1
		}
	}
	return 0
}

func RequestsNextAction(e elevator.Elevator) Action {
	switch e.dirn {
	case elevio.MD_Up:
		if RequestsAbove(e) {
			return Action{elevio.MD_Up, elevator.EB_Moving}
		} else if RequestsHere(e) {
			return Action{elevio.MD_Down, elevator.EB_DoorOpen}
		} else if RequestsBelow(e) {
			return Action{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return Action{elevio.MD_Stop, elevator.EB_Idle}
		}
	case elevio.MD_Down:
		if RequestsBelow(e) {
			return Action{elevio.MD_Down, elevator.EB_Moving}
		} else if RequestsHere(e) {
			return Action{elevio.MD_Up, elevator.EB_DoorOpen}
		} else if RequestsAbove(e) {
			return Action{elevio.MD_Up, elevator.EB_Moving}
		} else {
			return Action{elevio.MD_Stop, elevator.EB_Idle}
		}
	case elevio.MD_Stop:
		if RequestsHere(e) {
			return Action{elevio.MD_Stop, elevator.EB_DoorOpen}
		} else if RequestsAbove(e) {
			return Action{elevio.MD_Up, elevator.EB_Moving}
		} else if RequestsBelow(e) {
			return Action{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return Action{elevio.MD_Stop, elevator.EB_Idle}
		}
	default:
		return Action{elevio.MD_Stop, elevator.EB_Idle}
	}
}

func RequestShouldStop(e elevator.Elevator) int {
	switch e.dirn {
	case elevio.MD_Down:
		return e.requests[e.floor][elevio.BT_HallDown] || e.requests[e.floor][elevio.BT_Cab] || !RequestsBelow(e)
	case elevio.MD_Up:
		return e.requests[e.floor][elevio.BT_HallUp] || e.requests[e.floor][elevio.BT_Cab] || !Requestsabove(e)
	case elevio.MD_Stop:
		return 1
	default:
		return 1
	}
}

func ClearRequestImmediately(e elevator.Elevator, btnFloor int, btnType elevio.ButtonType) int {
	if e.floor == btnFloor {
		if e.dirn == elevio.MD_Up && btnType == elevio.BT_HallUp {
			return 1
		} else if e.dirn == elevio.MD_Down && btnType == elevio.BT_HallDown {
			return 1
		} else if e.dirn == elevio.MD_Stop {
			return 1
		} else if btnType == elevio.BT_Cab {
			return 1
		}
	return 0
}

//Clears all requests in the floor when the elevator stops
//This might have to be changed, assumes that everyone enters the elevator in the floor, regardless of direction
func ClearRequestCurrentFloor(e elevator.Elevator) {
	e.requests[e.floor][elevio.BT_Cab] = 0
	e.requests[e.floor][elevio.BT_HallUp] = 0
	e.requests[e.floor][elevio.BT_HallDown] = 0
}
