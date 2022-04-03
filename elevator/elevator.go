package elevator

import (
	"master/Driver-go/elevio"
)

type ElevBehaviour string

const (
	EB_Idle     ElevBehaviour = "idle"
	EB_Moving   ElevBehaviour = "moving"
	EB_DoorOpen ElevBehaviour = "doorOpen"
)

type Elev struct {
	Behaviour   ElevBehaviour          `json:"behaviour"`
	Floor       int                    `json:"floor"`
	Dirn        string                 `json:"direction"`
	CabRequests [elevio.NumFloors]bool `json:"cabRequests"`
}

type SingleElev struct {
	Floor     int
	Dirn      string
	Requests  [elevio.NumFloors][elevio.NumButtonTypes]bool
	Behaviour ElevBehaviour
}

func ElevUninitialized() Elev {
	uninitElev := Elev{
		Behaviour: EB_Idle,
		Floor:     -1,
		Dirn:      elevio.MotorDirToString(elevio.MD_Stop),
	}
	return uninitElev
}
