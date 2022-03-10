package main

import (
	"fmt"
	"master/Driver-go/elevio"
	"master/network/broadcast"
	"time"
)

//These structs will be JSON

type SlaveButtonEventMsg struct {
	Btn_floor int
	Btn_type  int
}
type MasterAckOrderMsg struct {
	Btn_floor int
	Btn_type  int
}

func main() {

	numFloors := 4
	elevio.Init("localhost:15659", numFloors)

	if elevio.GetFloor() == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
	}

	//fsm.SetAllLights(e)
	for floor := 0; floor < elevio.NumFloors; floor++ {
		for btn := 0; btn < elevio.NumButtonTypes; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, false)
		}
	}

	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)

	slaveButtonTx := make(chan SlaveButtonEventMsg)
	slaveFloorTx := make(chan int)
	slaveAckOrderDoneTx := make(chan bool)
	masterMotorDirRx := make(chan int)
	masterAckOrderRx := make(chan MasterAckOrderMsg) // burde lage en struct med button_type og floor
	masterTurnOffOrderLightRx := make(chan int)

	doorTimer := time.NewTimer(20 * time.Second)
	obstructionActive := false

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)
	go broadcast.Transmitter(16513, slaveButtonTx)
	go broadcast.Transmitter(16514, slaveFloorTx)
	go broadcast.Receiver(16515, masterMotorDirRx)
	go broadcast.Receiver(16516, masterAckOrderRx)
	go broadcast.Transmitter(16517, slaveAckOrderDoneTx)
	go broadcast.Receiver(16518, masterTurnOffOrderLightRx)

	//Testing
	//elevio.SetMotorDirection(elevio.MD_Up)
	//Testing

	doorOpen := false

	for {
		select {
		case a := <-drv_buttons:
			buttonEvent := SlaveButtonEventMsg{a.Floor, int(a.Button)} //Maybe a go routine
			slaveButtonTx <- buttonEvent
			fmt.Println("Floor")
			fmt.Println(buttonEvent.Btn_floor)
			fmt.Println("Button type")
			fmt.Println(buttonEvent.Btn_type)
			fmt.Println(" ")
		case a := <-drv_floors:
			floorEvent := a //Maybe a go routine
			slaveFloorTx <- floorEvent
			fmt.Println("Arrived at floor:")
			fmt.Println(floorEvent)
			fmt.Println("")

			if a == numFloors-1 {
				elevio.SetMotorDirection(elevio.MD_Stop)
			} else if a == 0 {
				elevio.SetMotorDirection(elevio.MD_Stop)
			}

		case a := <-drv_obstr:
			fmt.Printf("%+v\n", a)
			if a {
				obstructionActive = true
			} else {
				obstructionActive = false
				if doorOpen {
					doorTimer.Stop()
					doorTimer.Reset(3 * time.Second)
				}
			}

		case a := <-drv_stop:
			fmt.Printf("%+v\n", a)
			for f := 0; f < numFloors; f++ {
				for b := elevio.ButtonType(0); b < 3; b++ {
					elevio.SetButtonLamp(b, f, false)
				}
			}
		case <-doorTimer.C:
			if !obstructionActive {
				fmt.Println("Timed out")
				//fmt.Println(e.Behaviour)
				//e = fsm.Fsm_onDoorTimeout(e)
				elevio.SetDoorOpenLamp(false)
				doorOpen = false
			}

		case a := <-masterMotorDirRx:
			if a == 0 {
				elevio.SetMotorDirection(elevio.MD_Stop)
				fmt.Println("Elevator told to stop")
				elevio.SetDoorOpenLamp(true)
				doorOpen = true
				doorTimer.Stop()
				doorTimer.Reset(3 * time.Second)
				slaveAckOrderDoneTx <- true

			} else {
				elevio.SetMotorDirection(elevio.MotorDirection(a))
			}

		case a := <-masterAckOrderRx:
			elevio.SetButtonLamp(elevio.ButtonType(a.Btn_type), a.Btn_floor, true)

		case a := <-masterTurnOffOrderLightRx:
			elevio.SetButtonLamp(0, a, false)
			elevio.SetButtonLamp(1, a, false)
			elevio.SetButtonLamp(2, a, false)
		}
	}
}
