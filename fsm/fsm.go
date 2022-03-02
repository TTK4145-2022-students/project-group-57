package fsm

import (
	"master/Driver-go/elevio"
	"master/elevator"
	"master/requests"
	"time"
)

var timer1 time.Timer

func setAllLights(es elevator.Elevator) {
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

func Fsm_onRequestButtonPressed(e elevator.Elevator, btnFloor int, btn_type elevio.ButtonType) {

	switch e.Behaviour {
	case elevator.EB_DoorOpen:
		if requests.ClearRequestImmediately(e, btnFloor, btn_type) == 1 {
			timer1.Stop()
			timer1.Reset(3 * time.Second)
			go func() {
				<-timer1.C
				elevio.SetDoorOpenLamp(false)
			}()
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
			timer1.Stop()
			timer1.Reset(3 * time.Second)
			go func() {
				<-timer1.C
				elevio.SetDoorOpenLamp(false)
			}()

			e = requests.ClearRequestCurrentFloor(e)
		case elevator.EB_Moving:
			elevio.SetMotorDirection(e.Dirn)
		case elevator.EB_Idle:
		}
	}
}
