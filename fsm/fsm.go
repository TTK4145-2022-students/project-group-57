package fsm

import (
	"master/Driver-go/elevio"
	"master/elevator"
	"master/requests"
)

func setAllLights(es elevator.Elevator) {
	for floor := 0; floor < elevio.NumFloors; floor++ {
		for btn := 0; btn < elevio.NumButtonTypes; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, es.Requests) //kanskje [btn][floor]
		}
	}
}

func fsm_onInitBetweenFloors(e elevator.Elevator) {
	elevio.SetMotorDirection(elevio.MD_Down)
	e.Dirn = elevio.MD_Down //fix
	e.Behaviour = elevator.EB_Moving
}

func fsm_onRequestButtonPressed(e elevator.Elevator, btnFloor int, btn_type elevio.ButtonType) {

	switch e.Behaviour {
	case elevator.EB_DoorOpen:
		if requests.ClearRequestImmediately(e, btnFloor, btn_type) == 1 {
			//restart timer
		} else {
			e.Requests[btnFloor][btn_type] = 1
		}

	case elevator.EB_Moving:
		e.Requests[btnFloor][btn_type] = 1

	case elevator.EB_Idle:
		e.Requests[btnFloor][btn_type] = 1
		nextAction := requests.RequestsNextAction(e)
		e.Dirn = nextAction.Dirn
		e.Behaviour = nextAction.Behaviour
		switch nextAction.Behaviour {
		case elevator.EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			//start timer
			e.Requests = requests.ClearRequestCurrentFloor(e)
		case elevator.EB_Moving:
			elevio.SetMotorDirection(e.Dirn)
		case elevator.EB_Idle:
		}
	}
}
