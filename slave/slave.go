package main

import (
	"fmt"
	"master/Driver-go/elevio"
	"master/network/broadcast"
	"master/network/peers"
	"master/types"
	"os/exec"
	"time"
)

func main() {

	numFloors := 4
	elevio.Init("localhost:15660", numFloors)
	MyID := "one"

	if elevio.GetFloor() == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
	}

	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)

	commandDoorOpen := make(chan types.DoorOpen)
	slaveButtonTx := make(chan types.SlaveButtonEventMsg)
	slaveFloorTx := make(chan types.SlaveFloor, 5)
	masterMotorDirRx := make(chan types.MasterCommand)
	masterAckOrderRx := make(chan types.MasterAckOrderMsg) // burde lage en struct med button_type og floor
	masterSetOrderLight := make(chan types.SetOrderLight)
	slaveDoorOpened := make(chan types.DoorOpen)
	transmitEnable := make(chan bool)
	MasterMsg := make(chan types.MasterStruct)
	MasterInitStruct := make(chan types.MasterStruct)
	NewMasterIDCh := make(chan types.NewMasterID)

	go peers.Transmitter(16522, MyID, transmitEnable)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	go broadcast.Transmitter(16513, slaveButtonTx)
	go broadcast.Transmitter(16514, slaveFloorTx)
	go broadcast.Transmitter(16521, slaveDoorOpened)
	go broadcast.Transmitter(16527, MasterInitStruct)

	go broadcast.Receiver(16515, masterMotorDirRx)
	go broadcast.Receiver(16516, masterAckOrderRx)
	go broadcast.Receiver(16518, masterSetOrderLight)
	go broadcast.Receiver(16520, commandDoorOpen)
	go broadcast.Receiver(16523, MasterMsg)
	go broadcast.Receiver(16524, NewMasterIDCh)

	var MasterStruct types.MasterStruct
	//INIT
	err := exec.Command("gnome-terminal", "--", "go", "run", "../main.go", "init").Run()
	fmt.Println(err)
	//MasterStruct.CurrentMasterID = MyID

	doorTimer := time.NewTimer(100 * time.Second) //Trouble initializing timer like this, maybe
	doorIsOpen := false
	ObstructionActive := false
	fmt.Println(ObstructionActive)

	MasterTimeout := 5 * time.Second
	MasterTimer := time.NewTimer(MasterTimeout)
	for {
		select {
		case a := <-NewMasterIDCh:
			if a.SlaveID == MyID {
				MasterStruct.CurrentMasterID = a.NewMasterID
			}

		case a := <-MasterMsg:

			//ignore msg from another master
			if a.CurrentMasterID == MasterStruct.CurrentMasterID {
				MasterStruct = a
				fmt.Println(MasterStruct)
				MasterTimer.Stop()
				MasterTimer.Reset(MasterTimeout)
			}

		case <-MasterTimer.C:
			fmt.Println("Master is dead")
			fmt.Println("Last received message:")
			fmt.Println(MasterStruct)

			if MasterStruct.PeerList.Peers == nil {
				err := exec.Command("gnome-terminal", "--", "go", "run", "../main.go", "master").Run()
				fmt.Println(err)
				MasterStruct.CurrentMasterID = MyID
				go func(MasterStruct types.MasterStruct) {
					for i := 0; i < 10; i++ {
						MasterInitStruct <- MasterStruct
						time.Sleep(100 * time.Millisecond)
					}
				}(MasterStruct)
			} else if MasterStruct.PeerList.Peers[0] == MyID {
				err := exec.Command("gnome-terminal", "--", "go", "run", "../main.go", "master").Run()
				fmt.Println(err)
				go func(MasterStruct types.MasterStruct) {
					for i := 0; i < 10; i++ {
						MasterInitStruct <- MasterStruct
						time.Sleep(100 * time.Millisecond)
					}
				}(MasterStruct)
			} else {
				MasterTimer.Stop()
				MasterTimer.Reset(MasterTimeout)

			}

		case a := <-drv_buttons:
			buttonEvent := types.SlaveButtonEventMsg{
				ID:        MyID,
				Btn_floor: a.Floor,
				Btn_type:  int(a.Button)} //Maybe a go routine

			slaveButtonTx <- buttonEvent

		case a := <-drv_floors:
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
			if a {
				ObstructionActive = true
			} else {
				ObstructionActive = false
				if doorIsOpen {
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

		case a := <-masterMotorDirRx: //Recieve direction from master

			if a.ID == MyID {
				floor := elevio.GetFloor()
				if (floor == elevio.NumFloors-1 && a.Motordir == "up") ||
					(floor == 0 && a.Motordir == "down") {
					fmt.Println("Top or bottom floor")
					floorEvent := types.SlaveFloor{ID: MyID, NewFloor: floor}
					slaveFloorTx <- floorEvent
				} else {
					fmt.Println("Received dir")
					fmt.Println(a.Motordir)
					fmt.Println(doorIsOpen)
					if !doorIsOpen {
						fmt.Println("Door closed")
						elevio.SetMotorDirection(elevio.StringToMotorDir(a.Motordir))
					}
				}

			}

		case a := <-masterSetOrderLight:
			elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_HallUp), a.BtnFloor, a.LightOn[0])
			elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_HallDown), a.BtnFloor, a.LightOn[1])
			if a.ID == MyID {
				elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_Cab), a.BtnFloor, a.LightOn[2])
			}

		case a := <-commandDoorOpen:
			if a.ID == MyID && !doorIsOpen && elevio.GetFloor() != -1 {
				elevio.SetDoorOpenLamp(a.SetDoorOpen)
				if a.SetDoorOpen {
					doorIsOpen = true
					doorTimer.Stop()
					doorTimer.Reset(3 * time.Second)
				}
				slaveDoorOpened <- a
			}
		case <-doorTimer.C:
			if !ObstructionActive {
				doorIsOpen = false
				elevio.SetDoorOpenLamp(false)
				slaveDoorOpened <- types.DoorOpen{ID: MyID, SetDoorOpen: false}
			} else {
				doorTimer.Stop()
				doorTimer.Reset(3 * time.Second)
			}
		}
	}
}
