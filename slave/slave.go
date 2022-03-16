package main

import (
	"fmt"
	"master/Driver-go/elevio"
	"master/elevator"
	"master/fsm"
	"master/network/broadcast"
	"master/types"
)

//These structs will be JSON

func main() {

	numFloors := 4
	elevio.Init("localhost:15659", numFloors)

	e1 := elevator.Elevator{
		Behaviour:   elevator.EB_Idle,
		Floor:       elevio.GetFloor(),
		Dirn:        elevio.MotorDirToString(elevio.MD_Stop),
		CabRequests: [elevio.NumFloors]bool{},
	}

	if e1.Floor == -1 {
		e1 = fsm.Fsm_onInitBetweenFloors(e1)
	}

	fsm.SetAllLights(e1)

	//fsm.SetAllLights(e)
	/*for floor := 0; floor < elevio.NumFloors; floor++ {
		for btn := 0; btn < elevio.NumButtonTypes; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, false)
		}
	}*/

	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)

	commandDoorOpen := make(chan bool)
	stateChan := make(chan elevator.Elevator)
	slaveButtonTx := make(chan types.SlaveButtonEventMsg)
	slaveFloorTx := make(chan types.SlaveFloor)
	slaveAckOrderDoneTx := make(chan bool)
	masterMotorDirRx := make(chan string)
	masterAckOrderRx := make(chan types.MasterAckOrderMsg) // burde lage en struct med button_type og floor
	masterTurnOffOrderLightRx := make(chan int)
	slaveDoorOpened := make(chan bool)

	//obstructionActive := false

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	go broadcast.Transmitter(16513, slaveButtonTx)
	go broadcast.Transmitter(16514, slaveFloorTx)
	go broadcast.Transmitter(16517, slaveAckOrderDoneTx)
	go broadcast.Transmitter(16519, stateChan)
	go broadcast.Transmitter(16521, slaveDoorOpened)

	go broadcast.Receiver(16515, masterMotorDirRx)
	go broadcast.Receiver(16516, masterAckOrderRx)
	go broadcast.Receiver(16518, masterTurnOffOrderLightRx)
	go broadcast.Receiver(16520, commandDoorOpen)

	//doorOpen := false
	stateChan <- e1

	for {
		select {
		case a := <-drv_buttons:
			buttonEvent := types.SlaveButtonEventMsg{a.Floor, int(a.Button)} //Maybe a go routine
			slaveButtonTx <- buttonEvent
			fmt.Println("Floor")
			fmt.Println(buttonEvent.Btn_floor)
			fmt.Println("Button type")
			fmt.Println(buttonEvent.Btn_type)
			fmt.Println(" ")

		case a := <-drv_floors:
			//update local state
			/*floorEvent := a //Maybe a go routine
			slaveFloorTx <- floorEvent
			e1.Floor = a
			stateChan <- e1*/

			floorEvent := types.SlaveFloor{ID: "one", NewFloor: a}
			slaveFloorTx <- floorEvent

			fmt.Println("Arrived at floor:")
			fmt.Println(floorEvent.NewFloor)
			fmt.Println("")

			elevio.SetFloorIndicator(a)
			elevio.SetMotorDirection(elevio.MD_Stop)

			if a == numFloors-1 {
				elevio.SetMotorDirection(elevio.MD_Stop)
			} else if a == 0 {
				elevio.SetMotorDirection(elevio.MD_Stop)
			}

		case a := <-drv_obstr:
			fmt.Printf("%+v\n", a)
			/*if a {
				obstructionActive = true
			} else {
				obstructionActive = false
				if doorOpen {
					doorTimer.Stop()
					doorTimer.Reset(3 * time.Second)
				}
			}*/

		case a := <-drv_stop:
			fmt.Printf("%+v\n", a)
			for f := 0; f < numFloors; f++ {
				for b := elevio.ButtonType(0); b < 3; b++ {
					elevio.SetButtonLamp(b, f, false)
				}
			}

		case a := <-masterMotorDirRx: //Recieve direction from master
			e1.Dirn = a
			elevio.SetMotorDirection(elevio.StringToMotorDir(e1.Dirn))
			stateChan <- e1

		case a := <-masterAckOrderRx:
			elevio.SetButtonLamp(elevio.ButtonType(a.Btn_type), a.Btn_floor, true)

		case a := <-masterTurnOffOrderLightRx:
			elevio.SetButtonLamp(0, a, false)
			elevio.SetButtonLamp(1, a, false)
			elevio.SetButtonLamp(2, a, false)

		case a := <-commandDoorOpen:
			elevio.SetDoorOpenLamp(a)
			slaveDoorOpened <- a
		}
	}
}
