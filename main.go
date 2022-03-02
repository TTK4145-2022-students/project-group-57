package main

import (
	"fmt"
	"master/Driver-go/elevio"
	"master/elevator"
	"master/fsm"
	"time"
)

func main() {

	numFloors := 4
	elevio.Init("localhost:15657", numFloors)

	e := elevator.Elevator{
		Floor:     elevio.GetFloor(),
		Dirn:      elevio.MD_Stop,
		Requests:  [elevio.NumFloors][elevio.NumButtonTypes]bool{},
		Behaviour: elevator.EB_Idle,
	}

	if e.Floor == -1 {
		e = fsm.Fsm_onInitBetweenFloors(e)
	}

	fsm.SetAllLights(e)
	//var d elevio.MotorDirection = elevio.MD_Up
	//elevio.SetMotorDirection(d)

	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)

	doorTimer := time.NewTimer(20 * time.Second)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	for {
		select {
		case a := <-drv_buttons:
			fmt.Println("buttonevent")
			fmt.Printf("%+v\n", a)
			elevio.SetButtonLamp(a.Button, a.Floor, true)
			e = fsm.Fsm_onRequestButtonPressed(e, a.Floor, a.Button, doorTimer)

		case a := <-drv_floors:
			fmt.Println("floor")
			fmt.Printf("%+v\n", a)
			if a == numFloors-1 {
				e.Dirn = elevio.MD_Stop
			} else if a == 0 {
				e.Dirn = elevio.MD_Stop
			}
			e = fsm.Fsm_onFloorArrival(e, a, doorTimer)

		case a := <-drv_obstr:
			fmt.Printf("%+v\n", a)
			if a {
				elevio.SetMotorDirection(elevio.MD_Stop)
			} else {
				elevio.SetMotorDirection(e.Dirn)
			}

		case a := <-drv_stop:
			fmt.Printf("%+v\n", a)
			for f := 0; f < numFloors; f++ {
				for b := elevio.ButtonType(0); b < 3; b++ {
					elevio.SetButtonLamp(b, f, false)
				}
			}
		case <-doorTimer.C:
			fmt.Println("Timed out")
			fmt.Println(e.Behaviour)
			e = fsm.Fsm_onDoorTimeout(e)
		}
	}
}
