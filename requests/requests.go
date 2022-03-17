package requests

import (
	"master/Driver-go/elevio"
	"master/elevator"
)

type Action struct {
	Dirn      elevio.MotorDirection
	Behaviour elevator.ElevatorBehaviour
}

func RequestsAppendHallCab(hall [][2]bool, cab []bool) [elevio.NumFloors][elevio.NumButtonTypes]bool {
	var AllRequests [elevio.NumFloors][elevio.NumButtonTypes]bool
	for i, ele := range cab {
		AllRequests[i][0] = hall[i][0]
		AllRequests[i][1] = hall[i][1]
		AllRequests[i][2] = ele
	}
	return AllRequests
}

func RequestsAbove(e elevator.Elevator, reqs [elevio.NumFloors][elevio.NumButtonTypes]bool) bool {
	for i := e.Floor + 1; i < elevio.NumFloors; i++ {
		for btn := 0; btn < elevio.NumButtonTypes; btn++ {
			if reqs[i][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsBelow(e elevator.Elevator, reqs [elevio.NumFloors][elevio.NumButtonTypes]bool) bool {
	for i := 0; i < e.Floor; i++ {
		for btn := 0; btn < elevio.NumButtonTypes; btn++ {
			if reqs[i][btn] {
				return true
			}
		}
	}
	return false
}

//Modified to work without cabreqs
func RequestsHere(e elevator.Elevator, reqs [elevio.NumFloors][elevio.NumButtonTypes]bool) bool {
	for btn := 0; btn < elevio.NumButtonTypes; btn++ {
		if reqs[e.Floor][btn] {
			return true
		}
	}
	return false
}

func RequestsNextAction(e elevator.Elevator, reqs [elevio.NumFloors][elevio.NumButtonTypes]bool) Action {
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

func RequestShouldStop(e elevator.Elevator, reqs [elevio.NumFloors][elevio.NumButtonTypes]bool) bool {
	switch e.Dirn {
	case "down":
		return reqs[e.Floor][elevio.BT_HallDown] || reqs[e.Floor][elevio.BT_Cab] || !RequestsBelow(e, reqs)
	case "up":
		return reqs[e.Floor][elevio.BT_HallUp] || reqs[e.Floor][elevio.BT_Cab] || !RequestsAbove(e, reqs)
	case "stop":
		return true
	default:
		return true
	}
}

func ClearRequestCurrentFloor(e elevator.Elevator, reqs [elevio.NumFloors][elevio.NumButtonTypes]bool) (elevator.Elevator, [elevio.NumFloors][elevio.NumButtonTypes]bool) {
	//reqs.Requests[e.Floor][elevio.BT_Cab] = false
	//elevio.SetButtonLamp(elevio.BT_Cab, e.Floor, false)
	reqs[e.Floor][elevio.BT_Cab] = false
	reqs[e.Floor][elevio.BT_HallUp] = false
	//elevio.SetButtonLamp(elevio.BT_HallUp, e.Floor, false)
	reqs[e.Floor][elevio.BT_HallDown] = false
	//elevio.SetButtonLamp(elevio.BT_HallDown, e.Floor, false)
	return e, reqs
}
