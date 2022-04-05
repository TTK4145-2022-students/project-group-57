package types

import (
	"master/Driver-go/elevio"
	"master/elevator"
	"master/network/peers"
	"master/requests"
)

type SlaveButtonEventMsg struct {
	ID        string
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

/*
type AllRequests struct {
	Requests [][]bool
}

type HallRequests struct {
	Requests [][2]bool
}*/

type HRAInput struct {
	HallRequests [][2]bool                `json:"hallRequests"`
	States       map[string]elevator.Elev `json:"states"`
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

type AbleToMove struct {
	ID         string
	AbleToMove bool
}
type MySlaves struct {
	Active   []string
	Immobile []string
}

type MasterStruct struct {
	CurrentMasterID string
	MySlaves        MySlaves
	Isolated        bool
	Initialized     bool
	PeerList        peers.PeerUpdate
	HallRequests    [][elevio.NumButtonTypes - 1]bool
	ElevStates      map[string]elevator.Elev
}

type NewMasterID struct {
	SlaveID     string
	NewMasterID string
}
