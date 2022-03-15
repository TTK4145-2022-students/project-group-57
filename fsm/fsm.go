package fsm

import (
	"master/Driver-go/elevio"
	"master/elevator"
)

//Maybe change switch-case to something else?

//Modified: Resets all lamps, done in order to avoid requests from elevator as input
func SetAllLights(es elevator.Elevator) {
	for floor := 0; floor < elevio.NumFloors; floor++ {
		for btn := 0; btn < elevio.NumButtonTypes; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, false)
		}
	}
}

func Fsm_onInitBetweenFloors(e elevator.Elevator) elevator.Elevator {
	elevio.SetMotorDirection(elevio.MD_Down)
	e.Dirn = elevio.MotorDirToString(elevio.MD_Down)
	e.Behaviour = elevator.EB_Moving
	return e
}
