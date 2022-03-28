package main

import (
	"fmt"
	"master/Driver-go/elevio"
	"master/network/broadcast"
	"master/network/localip"
	"master/network/peers"
	"master/types"
	"time"
)

//These structs will be JSON

func main() {

	numFloors := 4
	elevio.Init("localhost:15659", numFloors)
	MyID := "one"

	/*e1 := elevator.Elevator{
		Floor:       elevio.GetFloor(),
		Dirn:        elevio.MotorDirToString(elevio.MD_Stop),
		Behaviour:   elevator.EB_Idle,
		CabRequests: [elevio.NumFloors]bool{},
	}*/

	if elevio.GetFloor() == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
	}

	//fsm.SetAllLights(e1)

	MyIP, _ := localip.LocalIP()
	fmt.Println(MyIP)

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

	commandDoorOpen := make(chan types.DoorOpen)
	slaveButtonTx := make(chan types.SlaveButtonEventMsg)
	slaveFloorTx := make(chan types.SlaveFloor)
	masterMotorDirRx := make(chan types.MasterCommand)
	masterAckOrderRx := make(chan types.MasterAckOrderMsg) // burde lage en struct med button_type og floor
	masterSetOrderLight := make(chan types.SetOrderLight)
	slaveDoorOpened := make(chan types.DoorOpen)
	transmitEnable := make(chan bool)

	go peers.Transmitter(16522, MyID, transmitEnable)

	//obstructionActive := false

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	go broadcast.Transmitter(16513, slaveButtonTx)
	go broadcast.Transmitter(16514, slaveFloorTx)
	go broadcast.Transmitter(16521, slaveDoorOpened)

	go broadcast.Receiver(16515, masterMotorDirRx)
	go broadcast.Receiver(16516, masterAckOrderRx)
	go broadcast.Receiver(16518, masterSetOrderLight)
	go broadcast.Receiver(16520, commandDoorOpen)

	doorTimer := time.NewTimer(100 * time.Second) //Trouble initializing timer like this, maybe
	doorIsOpen := false

	for {
		select {
		case a := <-drv_buttons:
			buttonEvent := types.SlaveButtonEventMsg{
				ID:        MyID,
				Btn_floor: a.Floor,
				Btn_type:  int(a.Button)} //Maybe a go routine

			slaveButtonTx <- buttonEvent

		case a := <-drv_floors:
			//update local state
			/*floorEvent := a //Maybe a go routine
			slaveFloorTx <- floorEvent
			e1.Floor = a
			stateChan <- e1*/

			floorEvent := types.SlaveFloor{ID: MyID, NewFloor: a}
			slaveFloorTx <- floorEvent

			elevio.SetFloorIndicator(a)
			elevio.SetMotorDirection(elevio.MD_Stop)

			if a == numFloors-1 {
				elevio.SetMotorDirection(elevio.MD_Stop)
			} else if a == 0 {
				elevio.SetMotorDirection(elevio.MD_Stop)
			}

		case a := <-drv_obstr:
			fmt.Printf("%+v\n", a)
			transmitEnable <- !a
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

			if a.ID == MyID {
				fmt.Println("Received dir")
				fmt.Println(a.Motordir)
				fmt.Println(doorIsOpen)
				if !doorIsOpen {
					fmt.Println("Door closed")
					elevio.SetMotorDirection(elevio.StringToMotorDir(a.Motordir))
				}

			}

		case a := <-masterSetOrderLight:
			elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_HallUp), a.BtnFloor, a.LightOn[0])
			elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_HallDown), a.BtnFloor, a.LightOn[1])
			if a.ID == MyID {
				elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_Cab), a.BtnFloor, a.LightOn[2])
			}

		case a := <-commandDoorOpen:
			if a.ID == MyID && !doorIsOpen {
				elevio.SetDoorOpenLamp(a.SetDoorOpen)
				if a.SetDoorOpen {
					doorIsOpen = true
					doorTimer.Stop()
					doorTimer.Reset(3 * time.Second)
				}
				slaveDoorOpened <- a
			}
		case <-doorTimer.C:
			doorIsOpen = false
			elevio.SetDoorOpenLamp(false)
			slaveDoorOpened <- types.DoorOpen{ID: MyID, SetDoorOpen: false}
		}
	}
}
