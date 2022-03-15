package requests

import (
	"master/Driver-go/elevio"
	"master/elevator"
)

type Action struct {
	Dirn      elevio.MotorDirection
	Behaviour elevator.ElevatorBehaviour
}
type AllRequests struct {
	Requests [][]bool
}

type MasterHallRequests struct {
	Requests [elevio.NumFloors][2]bool
}

func RequestsAbove(e elevator.Elevator, reqs MasterHallRequests) bool {
	for i := e.Floor + 1; i < elevio.NumFloors; i++ {
		for btn := 0; btn < 2; btn++ {
			if reqs.Requests[i][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsBelow(e elevator.Elevator, reqs MasterHallRequests) bool {
	for i := 0; i < e.Floor; i++ {
		for btn := 0; btn < 2; btn++ {
			if reqs.Requests[i][btn] {
				return true
			}
		}
	}
	return false
}

//Modified to work without cabreqs
func RequestsHere(e elevator.Elevator, reqs MasterHallRequests) bool {
	for btn := 0; btn < 2; btn++ {
		if reqs.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

func RequestsNextAction(e elevator.Elevator, reqs MasterHallRequests) Action {
	switch e.Dirn {
	case "up":
		if RequestsAbove(e, reqs) {
			return Action{elevio.MD_Up, elevator.EB_Moving}
		} else if RequestsHere(e, reqs) {
			return Action{elevio.MD_Down, elevator.EB_DoorOpen}
		} else if RequestsBelow(e, reqs) {
			return Action{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return Action{elevio.MD_Stop, elevator.EB_Idle}
		}
	case "down":
		if RequestsBelow(e, reqs) {
			return Action{elevio.MD_Down, elevator.EB_Moving}
		} else if RequestsHere(e, reqs) {
			return Action{elevio.MD_Up, elevator.EB_DoorOpen}
		} else if RequestsAbove(e, reqs) {
			return Action{elevio.MD_Up, elevator.EB_Moving}
		} else {
			return Action{elevio.MD_Stop, elevator.EB_Idle}
		}
	case "stop":
		if RequestsHere(e, reqs) {
			return Action{elevio.MD_Stop, elevator.EB_DoorOpen}
		} else if RequestsAbove(e, reqs) {
			return Action{elevio.MD_Up, elevator.EB_Moving}
		} else if RequestsBelow(e, reqs) {
			return Action{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return Action{elevio.MD_Stop, elevator.EB_Idle}
		}
	default:
		return Action{elevio.MD_Stop, elevator.EB_Idle}
	}
}

func RequestShouldStop(e elevator.Elevator, reqs MasterHallRequests) bool {
	switch e.Dirn {
	case "down":
		return reqs.Requests[e.Floor][elevio.BT_HallDown] || !RequestsBelow(e, reqs)
	case "up":
		return reqs.Requests[e.Floor][elevio.BT_HallUp] || !RequestsAbove(e, reqs)
	case "stop":
		return true
	default:
		return true
	}
}

func ClearRequestCurrentFloor(e elevator.Elevator, reqs MasterHallRequests) (elevator.Elevator, MasterHallRequests) {
	//reqs.Requests[e.Floor][elevio.BT_Cab] = false
	//elevio.SetButtonLamp(elevio.BT_Cab, e.Floor, false)
	reqs.Requests[e.Floor][elevio.BT_HallUp] = false
	//elevio.SetButtonLamp(elevio.BT_HallUp, e.Floor, false)
	reqs.Requests[e.Floor][elevio.BT_HallDown] = false
	//elevio.SetButtonLamp(elevio.BT_HallDown, e.Floor, false)
	return e, reqs
}
