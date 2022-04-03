package requests

import (
	"fmt"
	"master/Driver-go/elevio"
	"master/elevator"
)

type Action struct {
	Dirn      elevio.MotorDirection
	Behaviour elevator.ElevBehaviour
}

func RequestsAppendHallCab(hall [][2]bool, cab [4]bool) [elevio.NumFloors][elevio.NumButtonTypes]bool {
	var AllRequests [elevio.NumFloors][elevio.NumButtonTypes]bool
	for i, ele := range cab {
		AllRequests[i][0] = hall[i][0]
		AllRequests[i][1] = hall[i][1]
		AllRequests[i][2] = ele
	}
	return AllRequests
}

func RequestsSplitHallCab(reqs [elevio.NumFloors][elevio.NumButtonTypes]bool) ([][2]bool, [4]bool) {
	HallRequests := [][2]bool{{false, false}, {false, false}, {false, false}, {false, false}}
	var CabRequests [4]bool
	fmt.Println("Reqs:")
	fmt.Println(reqs)
	for i := 0; i < elevio.NumFloors; i++ {
		fmt.Println("i:")
		fmt.Println(i)
		for j := 0; j < elevio.NumButtonTypes-1; j++ {
			fmt.Println("j:")
			fmt.Println(j)
			HallRequests[i][j] = reqs[i][j]
		}
		CabRequests[i] = reqs[i][2]
	}
	return HallRequests, CabRequests
}

func RequestsAbove(e elevator.Elev, reqs [elevio.NumFloors][elevio.NumButtonTypes]bool) bool {
	for i := e.Floor + 1; i < elevio.NumFloors; i++ {
		for btn := 0; btn < elevio.NumButtonTypes; btn++ {
			if reqs[i][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsBelow(e elevator.Elev, reqs [elevio.NumFloors][elevio.NumButtonTypes]bool) bool {
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
func RequestsHere(e elevator.Elev, reqs [elevio.NumFloors][elevio.NumButtonTypes]bool) bool {
	for btn := 0; btn < elevio.NumButtonTypes; btn++ {
		if reqs[e.Floor][btn] {
			return true
		}
	}
	return false
}

func RequestsNextAction(e elevator.Elev, reqs [elevio.NumFloors][elevio.NumButtonTypes]bool) Action {
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

func RequestShouldStop(e elevator.Elev, reqs [elevio.NumFloors][elevio.NumButtonTypes]bool) bool {
	switch e.Dirn {
	case "down":
		return reqs[e.Floor][elevio.BT_HallDown] || reqs[e.Floor][elevio.BT_Cab] || !RequestsBelow(e, reqs)
	case "up":
		return reqs[e.Floor][elevio.BT_HallUp] || reqs[e.Floor][elevio.BT_Cab] || !RequestsAbove(e, reqs)
	case "stop":
		return false
	default:
		return false
	}
}

func ClearRequestCurrentFloor(e elevator.Elev, reqs [elevio.NumFloors][elevio.NumButtonTypes]bool) (elevator.Elev, [elevio.NumFloors][elevio.NumButtonTypes]bool) {
	//reqs.Requests[e.Floor][elevio.BT_Cab] = false
	//elevio.SetButtonLamp(elevio.BT_Cab, e.Floor, false)
	reqs[e.Floor][elevio.BT_Cab] = false
	reqs[e.Floor][elevio.BT_HallUp] = false
	//elevio.SetButtonLamp(elevio.BT_HallUp, e.Floor, false)
	reqs[e.Floor][elevio.BT_HallDown] = false
	//elevio.SetButtonLamp(elevio.BT_HallDown, e.Floor, false)
	return e, reqs
}

func ShouldClearHallRequest(e elevator.Elev, Hallreqs [][2]bool) [2]bool {
	AllReqs := RequestsAppendHallCab(Hallreqs, e.CabRequests)
	if Hallreqs[e.Floor][0] && Hallreqs[e.Floor][1] {
		switch e.Dirn {
		case "up":
			if !RequestsAbove(e, AllReqs) && !AllReqs[e.Floor][elevio.BT_HallUp] {
				return [2]bool{false, false}
			} else if AllReqs[e.Floor][elevio.BT_HallDown] {
				return [2]bool{false, true}
			}
		case "down":
			if !RequestsBelow(e, AllReqs) && !AllReqs[e.Floor][elevio.BT_HallDown] {
				return [2]bool{false, false}
			} else if AllReqs[e.Floor][elevio.BT_HallUp] {
				return [2]bool{true, false}
			}
		}
	}
	return [2]bool{false, false}
}

func SingleElevRequestShouldStop(e elevator.Elev, reqs [elevio.NumFloors][elevio.NumButtonTypes]bool) bool {
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

func ClearRequestImmediately(e elevator.Elev, btnFloor int, btnType elevio.ButtonType) int {
	if e.Floor == btnFloor {
		if e.Dirn == "up" && btnType == elevio.BT_HallUp {
			return 1
		} else if e.Dirn == "down" && btnType == elevio.BT_HallDown {
			return 1
		} else if e.Dirn == "stop" {
			return 1
		} else if btnType == elevio.BT_Cab {
			return 1
		}
		return 0
	}
	return 0
}
