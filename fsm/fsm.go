package fsm

import (
	"fmt"
	"master/Driver-go/elevio"
	"master/elevator"
	"master/requests"
	"time"
)

//Maybe change switch-case to something else?

func SetAllLights(Allreqs [elevio.NumFloors][elevio.NumButtonTypes]bool) {
	for floor := 0; floor < elevio.NumFloors; floor++ {
		for btn := 0; btn < elevio.NumButtonTypes; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, Allreqs[floor][btn])
		}
	}
}

func SetOnlyHallLights(Allreqs [elevio.NumFloors][elevio.NumButtonTypes]bool) {
	for floor := 0; floor < elevio.NumFloors; floor++ {
		for btn := 0; btn < elevio.NumButtonTypes-1; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, Allreqs[floor][btn])
		}
	}
}

func Fsm_onInitBetweenFloors(e elevator.Elev) elevator.Elev {
	elevio.SetMotorDirection(elevio.MD_Down)
	e.Dirn = elevio.MotorDirToString(elevio.MD_Down)
	e.Behaviour = elevator.EB_Moving
	return e
}

func UnInitializedElev() elevator.Elev {
	var e elevator.Elev
	e.Floor = 0
	e.Dirn = "stop"
	e.Behaviour = "idle"
	e.CabRequests = [elevio.NumFloors]bool{}
	return e

}

func Fsm_onRequestButtonPressed(e elevator.Elev, reqs [elevio.NumFloors][elevio.NumButtonTypes]bool, btnFloor int, btn_type elevio.ButtonType, doorTimer *time.Timer) (elevator.Elev, [elevio.NumFloors][elevio.NumButtonTypes]bool) {

	switch e.Behaviour {
	case elevator.EB_DoorOpen:
		if requests.ClearRequestImmediately(e, btnFloor, btn_type) == 1 {
			elevio.SetDoorOpenLamp(true)
			doorTimer.Stop()
			doorTimer.Reset(3 * time.Second)
		} else {
			reqs[btnFloor][btn_type] = true
		}

	case elevator.EB_Moving:
		reqs[btnFloor][btn_type] = true

	case elevator.EB_Idle:
		reqs[btnFloor][btn_type] = true
		nextAction := requests.RequestsNextAction(e, reqs)
		e.Dirn = elevio.MotorDirToString(nextAction.Dirn)
		e.Behaviour = nextAction.Behaviour
		switch nextAction.Behaviour {
		case elevator.EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			doorTimer.Stop()
			doorTimer.Reset(3 * time.Second)
			e, reqs = requests.ClearRequestCurrentFloor(e, reqs)
		case elevator.EB_Moving:
			elevio.SetMotorDirection(elevio.StringToMotorDir(e.Dirn))
		case elevator.EB_Idle:
		}
	}
	SetAllLights(reqs)
	return e, reqs
}

func Fsm_onFloorArrival(e elevator.Elev, reqs [elevio.NumFloors][elevio.NumButtonTypes]bool, newFloor int, doorTimer *time.Timer) (elevator.Elev, [elevio.NumFloors][elevio.NumButtonTypes]bool) {
	e.Floor = newFloor
	elevio.SetFloorIndicator(newFloor)
	fmt.Println(e.Behaviour)
	switch e.Behaviour {
	case elevator.EB_Moving:
		fmt.Println("inside case moving")
		if requests.RequestShouldStop(e, reqs) {
			fmt.Println("trying to stop")
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			doorTimer.Stop()
			doorTimer.Reset(3 * time.Second)
			fmt.Println("Timer started")
			//e, reqs = requests.ClearRequestCurrentFloor(e, reqs) //NO
			HallRequests, _ := requests.RequestsSplitHallCab(reqs)
			ClearHallReqs := requests.ShouldClearHallRequest(e, HallRequests)
			reqs[e.Floor][0] = ClearHallReqs[0]
			reqs[e.Floor][1] = ClearHallReqs[1]
			reqs[e.Floor][2] = false
			SetAllLights(reqs)
			e.Behaviour = elevator.EB_DoorOpen
		}
	}
	return e, reqs
}

func Fsm_onDoorTimeout(e elevator.Elev,
	reqs [elevio.NumFloors][elevio.NumButtonTypes]bool) (elevator.Elev, [elevio.NumFloors][elevio.NumButtonTypes]bool) {
	elevio.SetDoorOpenLamp(false)
	switch e.Behaviour {
	case elevator.EB_DoorOpen:
		a := requests.RequestsNextAction(e, reqs)
		e.Dirn = elevio.MotorDirToString(a.Dirn)
		e.Behaviour = a.Behaviour

		fmt.Println("Next", e.Behaviour)
		fmt.Println("dir", e.Dirn)

		switch e.Behaviour {
		case elevator.EB_DoorOpen:
			e, reqs = requests.ClearRequestCurrentFloor(e, reqs)
			SetAllLights(reqs)
		case elevator.EB_Moving:
			elevio.SetMotorDirection(elevio.StringToMotorDir(e.Dirn))
		case elevator.EB_Idle:
			elevio.SetMotorDirection(elevio.StringToMotorDir(e.Dirn))
			return e, reqs
		}
	}
	return e, reqs
}
