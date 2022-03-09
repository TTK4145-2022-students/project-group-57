package fsm

import (
	"fmt"
	"master/Driver-go/elevio"
	"master/elevator"
	"master/requests"
	"time"
)

//Maybe change switch-case to something else?

func SetAllLights(es elevator.Elevator) {
	for floor := 0; floor < elevio.NumFloors; floor++ {
		for btn := 0; btn < elevio.NumButtonTypes; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, es.Requests[floor][btn])
		}
	}
}

func Fsm_onInitBetweenFloors(e elevator.Elevator) elevator.Elevator {
	elevio.SetMotorDirection(elevio.MD_Down)
	e.Dirn = elevio.MD_Down
	e.Behaviour = elevator.EB_Moving
	return e
}

func Fsm_onRequestButtonPressed(e elevator.Elevator, btnFloor int, btn_type elevio.ButtonType, doorTimer *time.Timer) elevator.Elevator {

	switch e.Behaviour {
	case elevator.EB_DoorOpen:
		if requests.ClearRequestImmediately(e, btnFloor, btn_type) == 1 {
			elevio.SetDoorOpenLamp(true)
			doorTimer.Stop()
			doorTimer.Reset(3 * time.Second)
		} else {
			e.Requests[btnFloor][btn_type] = true
		}

	case elevator.EB_Moving:
		e.Requests[btnFloor][btn_type] = true

	case elevator.EB_Idle:
		e.Requests[btnFloor][btn_type] = true
		nextAction := requests.RequestsNextAction(e)
		e.Dirn = nextAction.Dirn
		e.Behaviour = nextAction.Behaviour
		switch nextAction.Behaviour {
		case elevator.EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			doorTimer.Stop()
			doorTimer.Reset(3 * time.Second)
			e = requests.ClearRequestCurrentFloor(e)
		case elevator.EB_Moving:
			elevio.SetMotorDirection(e.Dirn)
		case elevator.EB_Idle:
		}
	}
	SetAllLights(e)
	return e
}

func Fsm_onFloorArrival(e elevator.Elevator, newFloor int, doorTimer *time.Timer) elevator.Elevator {
	e.Floor = newFloor
	elevio.SetFloorIndicator(newFloor)
	fmt.Println(e.Behaviour)
	switch e.Behaviour {
	case elevator.EB_Moving:
		fmt.Println("inside case moving")
		if requests.RequestShouldStop(e) {
			fmt.Println("trying to stop")
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			doorTimer.Stop()
			doorTimer.Reset(3 * time.Second)
			fmt.Println("Timer started")
			e = requests.ClearRequestCurrentFloor(e)
			SetAllLights(e)
			e.Behaviour = elevator.EB_DoorOpen
		}
	}
	return e
}

func Fsm_onDoorTimeout(e elevator.Elevator) elevator.Elevator {
	elevio.SetDoorOpenLamp(false)
	switch e.Behaviour {
	case elevator.EB_DoorOpen:
		a := requests.RequestsNextAction(e)
		e.Dirn = a.Dirn
		e.Behaviour = a.Behaviour

		fmt.Println("Next", e.Behaviour)
		fmt.Println("dir", e.Dirn)

		switch e.Behaviour {
		case elevator.EB_DoorOpen:
			e = requests.ClearRequestCurrentFloor(e)
			SetAllLights(e)
		case elevator.EB_Moving:
			elevio.SetMotorDirection(e.Dirn)
		case elevator.EB_Idle:
			elevio.SetMotorDirection(e.Dirn)
			return e
		}
	}
	return e
}
