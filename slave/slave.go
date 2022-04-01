package main

import (
	"fmt"
	"master/Driver-go/elevio"
	"master/elevator"
	"master/fsm"
	"master/network/broadcast"
	"master/network/peers"
	"master/requests"
	"master/types"
	"os/exec"
	"strconv"
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
	PeerUpdateCh := make(chan peers.PeerUpdate)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	go broadcast.Transmitter(16513, slaveButtonTx)
	go broadcast.Transmitter(16514, slaveFloorTx)
	go broadcast.Transmitter(16521, slaveDoorOpened)
	go broadcast.Transmitter(16527, MasterInitStruct)
	go peers.Transmitter(16522, MyID, transmitEnable)

	go broadcast.Receiver(16515, masterMotorDirRx)
	go broadcast.Receiver(16516, masterAckOrderRx)
	go broadcast.Receiver(16518, masterSetOrderLight)
	go broadcast.Receiver(16520, commandDoorOpen)
	go broadcast.Receiver(16523, MasterMsg)
	go broadcast.Receiver(16524, NewMasterIDCh)
	go peers.Receiver(16522, PeerUpdateCh)

	var MasterStruct types.MasterStruct
	var Peerlist peers.PeerUpdate
	//INIT
	elevio.SetDoorOpenLamp(false)
	floor := elevio.GetFloor()
	if floor == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
		floor = <-drv_floors
	}
	elevio.SetMotorDirection(elevio.MD_Stop)
	elevio.SetFloorIndicator(floor)

	CurrentFloor := strconv.Itoa(floor)

	//Send ack/alive?
	err := exec.Command("gnome-terminal", "--", "go", "run", "../main.go", "init", MyID, CurrentFloor).Run()
	fmt.Println(err)

	//MasterStruct.CurrentMasterID = MyID

	doorTimer := time.NewTimer(100 * time.Second) //Trouble initializing timer like this, maybe
	doorIsOpen := false
	obstructionActive := false
	fmt.Println(obstructionActive)

	MasterTimeout := 5 * time.Second
	MasterTimer := time.NewTimer(MasterTimeout)
	for {
		select {
		case NewPeerlist := <-PeerUpdateCh:
			Peerlist = NewPeerlist

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
			fmt.Println(len(Peerlist.Peers))

			//Single elevator
			if len(Peerlist.Peers) == 0 {
				fmt.Println("single")
				HallRequests := MasterStruct.HRAInput.HallRequests
				fmt.Println(len(HallRequests))
				if len(HallRequests) == 0 {
					HallRequests = [][2]bool{{false, false}, {false, false}, {false, false}, {false, false}}
				}
				CabRequests := MasterStruct.HRAInput.States[MyID].CabRequests
				if len(CabRequests) == 0 {
					CabRequests = [4]bool{false, false, false, false}
				}
				SingleElevatorRequests := requests.RequestsAppendHallCab(HallRequests, CabRequests)

				e := elevator.Elevator{
					Behaviour:   elevator.EB_Idle,
					Floor:       elevio.GetFloor(),
					Dirn:        "stop",
					CabRequests: CabRequests,
				}
				if e.Floor == -1 {
					e = fsm.Fsm_onInitBetweenFloors(e)
				}
				for len(Peerlist.Peers) == 1 { //must be zero
					select {
					case a := <-drv_buttons:
						fmt.Println("buttonevent")
						fmt.Printf("%+v\n", a)
						elevio.SetButtonLamp(a.Button, a.Floor, true)
						e, SingleElevatorRequests = fsm.Fsm_onRequestButtonPressed(e, SingleElevatorRequests, a.Floor, a.Button, doorTimer)

					case a := <-drv_floors:
						fmt.Println("floor")
						fmt.Printf("%+v\n", a)
						if a == numFloors-1 {
							e.Dirn = "stop"
						} else if a == 0 {
							e.Dirn = "stop"
						}
						e, SingleElevatorRequests = fsm.Fsm_onFloorArrival(e, SingleElevatorRequests, a, doorTimer)

					case a := <-drv_obstr: //Ask: Obstruction between floors, three seconds with open door after obstruction off.?
						fmt.Printf("%+v\n", a)
						if a {
							obstructionActive = true
						} else {
							obstructionActive = false
							if e.Behaviour == elevator.EB_DoorOpen {
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
							fmt.Println(e.Behaviour)
							e, SingleElevatorRequests = fsm.Fsm_onDoorTimeout(e, SingleElevatorRequests)
						}
					case a := <-PeerUpdateCh:
						Peerlist = a
						HallRequests, CabRequests = requests.RequestsSplitHallCab(SingleElevatorRequests)
						e.CabRequests = CabRequests
						MasterStruct.HRAInput.States[MyID] = e
						MasterStruct.HRAInput.HallRequests = HallRequests

					}
				}

			} else if Peerlist.Peers[0] == MyID { //I am master
				err := exec.Command("gnome-terminal", "--", "go", "run", "../main.go", "master").Run()
				fmt.Println(err)
				go func(MasterStruct types.MasterStruct) {
					for i := 0; i < 10; i++ {
						MasterInitStruct <- MasterStruct
						time.Sleep(100 * time.Millisecond)
					}
				}(MasterStruct)
			} else { //Somebody else is master
				MasterTimer.Stop()
				MasterTimer.Reset(MasterTimeout)
			}
			//Two cases:
			//1. Master is dead, start new master (contact with other elevators) -> start new master
			//2. Lost network -> become single elevator

			//state: isolated
			//Single elevator mode (exercise 4)?
			//Use last received struct

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
				obstructionActive = true
			} else {
				obstructionActive = false
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
			if !obstructionActive {
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
