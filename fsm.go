package fsm

import (
	"Driver-go/elevio"
	"elevator"
)


func setAllLights(Elevator es){
	for floor :=0; floor < _numFloors; floor++ {
		for btn := 0; btn < _numButtonTypes; btn++{
			elevio.SetButtonLamp(btn, floor, es.requests[floor][btn]) //kanskje [btn][floor]
		}
	} 
}

func fsm_onInitBetweenFloors(){
	elevio.SetMotorDirection(D_Down)
	elevator.dirn = D_Down
	elevator.behaviour = EB_Moving
}

func fsm_onRequestButtonPressed(btnFloor int, btn_type Button){

	switch elevator.behaviour {
	case EB_DoorOpen:
		if requestShouldClearImmediately(elevator, btn_floor, btn_type){
			//restart timer 
		}
		else{
			elevator.requests[btn_floor][btn_type] = 1 
		}

	case EB_Moving:
		elevator.requests[btn_floor][btn_type] = 1 
	
	case EB_Idle:
		elevator.requests[btn_floor][btn_type] = 1 
		nextAction := RequestsNextAction(elevator)
		elevator.dirn = nextAction.dirn
		elevator.behaviour = nextAction.behaviour
		switch nextAction.behaviour {
		case EB_DoorOpen:
			elevio.SetDoorOpenLamp(1)
			//start timer
			elevator = requests_clearAtCurrentFloor(elevator)
		case EB_Moving:
			elevio.SetMotorDirection(elevator.dirn)
		case EB_Idle:
		}
	}
}