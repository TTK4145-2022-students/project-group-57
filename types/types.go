package types

import (
	"master/Driver-go/elevio"
	"master/elevator"
)

type SlaveButtonEventMsg struct {
	Btn_floor int
	Btn_type  int
}
type MasterAckOrderMsg struct {
	Btn_floor int
	Btn_type  int
}

type SlaveFloor struct {
	ID       string
	NewFloor int
}

type AllRequests struct {
	Requests [][]bool
}

type MasterHallRequests struct {
	Requests [elevio.NumFloors][2]bool
}

type HRAInput struct {
	HallRequests [elevio.NumFloors][2]bool    `json:"hallRequests"`
	States       map[string]elevator.Elevator `json:"states"`
}
