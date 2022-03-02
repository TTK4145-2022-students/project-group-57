package requests

import (
	"master/Driver-go/elevio"
	"master/elevator"
)

type Action struct {
	Dirn      elevio.MotorDirection
	Behaviour elevator.ElevatorBehaviour
}

func RequestsAbove(e elevator.Elevator) bool {
	for i := e.Floor + 1; i < elevio.NumFloors; i++ {
		for btn := 0; btn < elevio.NumButtonTypes; btn++ {
			if e.Requests[i][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsBelow(e elevator.Elevator) bool {
	for i := 0; i < e.Floor; i++ {
		for btn := 0; btn < elevio.NumButtonTypes; btn++ {
			if e.Requests[i][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsHere(e elevator.Elevator) bool {
	for btn := 0; btn < elevio.NumButtonTypes; btn++ {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

func RequestsNextAction(e elevator.Elevator) Action {
	switch e.Dirn {
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

func RequestShouldStop(e elevator.Elevator) bool {
	switch e.Dirn {
	case elevio.MD_Down:
		return e.Requests[e.Floor][elevio.BT_HallDown] || e.Requests[e.Floor][elevio.BT_Cab] || !RequestsBelow(e)
	case elevio.MD_Up:
		return e.Requests[e.Floor][elevio.BT_HallUp] || e.Requests[e.Floor][elevio.BT_Cab] || !RequestsAbove(e)
	case elevio.MD_Stop:
		return true
	default:
		return true
	}
}

func ClearRequestImmediately(e elevator.Elevator, btnFloor int, btnType elevio.ButtonType) int {
	if e.Floor == btnFloor {
		if e.Dirn == elevio.MD_Up && btnType == elevio.BT_HallUp {
			return 1
		} else if e.Dirn == elevio.MD_Down && btnType == elevio.BT_HallDown {
			return 1
		} else if e.Dirn == elevio.MD_Stop {
			return 1
		} else if btnType == elevio.BT_Cab {
			return 1
		}
		return 0
	}
	return 0
}

//Clears all requests in the floor when the elevator stops
//This might have to be changed, assumes that everyone enters the elevator in the floor, regardless of direction
func ClearRequestCurrentFloor(e elevator.Elevator) elevator.Elevator {
	e.Requests[e.Floor][elevio.BT_Cab] = false
	e.Requests[e.Floor][elevio.BT_HallUp] = false
	e.Requests[e.Floor][elevio.BT_HallDown] = false
	return e
}