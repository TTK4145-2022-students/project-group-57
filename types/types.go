package types

import (
	"master/elevator"
	"master/requests"
)

type SlaveButtonEventMsg struct {
	ID        string
	Btn_floor int
	Btn_type  int
}
type MasterAckOrderMsg struct {
	Btn_floor int
	Btn_type  int
}

type DoorOpen struct {
	ID          string
	SetDoorOpen bool
}

type MasterCommand struct {
	ID       string
	Motordir string
}

type SlaveFloor struct {
	ID       string
	NewFloor int
}

type AllRequests struct {
	Requests [][]bool
}

type MasterHallRequests struct {
	Requests [][2]bool
}

type ElevatorHallRequests struct {
	Requests [][2]bool
}

type HRAInput struct {
	HallRequests [][2]bool                    `json:"hallRequests"`
	States       map[string]elevator.Elevator `json:"states"`
}

type SetOrderLight struct {
	ID       string
	BtnFloor int
	LightOn  [3]bool
}

type NewAction struct {
	ID     string
	Action requests.Action
}
