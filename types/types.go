package types

import (
	"master/elevator"
	"master/network/peers"
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
	MasterID    string
	ID          string
	SetDoorOpen bool
}

type MasterCommand struct {
	MasterID string
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
	MasterID string
	ID       string
	LightOn  [4][3]bool
}

type NewAction struct {
	ID     string
	Action requests.Action
}

type MasterStruct struct {
	CurrentMasterID string
	MySlaves        []string
	Isolated        bool
	AlreadyExists   bool
	PeerList        peers.PeerUpdate
	HRAInput        HRAInput
}

type NewMasterID struct {
	SlaveID     string
	NewMasterID string
}
