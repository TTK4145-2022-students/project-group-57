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
	elevio.Init("localhost:15661", numFloors)
	//Can use localIP, but not when testing on single computer
	MyID := "one"

	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)

	commandDoorOpen := make(chan types.DoorOpen)
	slaveButtonTx := make(chan types.SlaveButtonEventMsg)
	slaveFloorTx := make(chan types.SlaveFloor, 5)
	masterMotorDirRx := make(chan types.MasterCommand)
	masterSetOrderLight := make(chan types.SetOrderLight)
	slaveDoorOpened := make(chan types.DoorOpen)
	transmitEnable := make(chan bool)
	MasterMsg := make(chan types.MasterStruct)
	MasterInitStruct := make(chan types.MasterStruct)
	NewMasterIDCh := make(chan types.NewMasterID)
	PeerUpdateCh := make(chan peers.PeerUpdate)
	AbleToMoveCh := make(chan types.AbleToMove)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	go broadcast.Transmitter(16513, slaveButtonTx)
	go broadcast.Transmitter(16514, slaveFloorTx)
	go broadcast.Transmitter(16521, slaveDoorOpened)
	go broadcast.Transmitter(16527, MasterInitStruct)
	go broadcast.Transmitter(16528, AbleToMoveCh)

	go broadcast.Receiver(16515, masterMotorDirRx)
	go broadcast.Receiver(16518, masterSetOrderLight)
	go broadcast.Receiver(16520, commandDoorOpen)
	go broadcast.Receiver(16523, MasterMsg)
	go broadcast.Receiver(16524, NewMasterIDCh)

	go peers.Transmitter(16529, MyID, transmitEnable)
	go peers.Receiver(16529, PeerUpdateCh)

	//INIT
	var Peerlist peers.PeerUpdate
	InitLightsOff := [4][3]bool{}
	fsm.SetAllLights(InitLightsOff)
	elevio.SetDoorOpenLamp(false)
	floor := elevio.GetFloor()
	if floor == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
		floor = <-drv_floors
	}
	elevio.SetMotorDirection(elevio.MD_Stop)
	elevio.SetFloorIndicator(floor)
	CurrentFloor := strconv.Itoa(floor)

	err := exec.Command("gnome-terminal", "--", "go", "run", "../main.go", "init", MyID, CurrentFloor).Run()
	fmt.Println(err)

	MySlaves := types.MySlaves{Active: []string{MyID}}

	MasterStruct := types.MasterStruct{
		CurrentMasterID: MyID,
		Isolated:        false,
		Initialized:     false,
		PeerList:        peers.PeerUpdate{},
		HallRequests:    [][2]bool{{false, false}, {false, false}, {false, false}, {false, false}},
		ElevStates:      map[string]elevator.Elev{},
		MySlaves:        MySlaves,
	}

	e := fsm.UnInitializedElev()
	e.Floor = floor
	MasterStruct.ElevStates[MyID] = e

	//MasterStruct.CurrentMasterID = MyID

	doorTimer := time.NewTimer(100 * time.Second) //Trouble initializing timer like this, maybe
	doorIsOpen := false
	obstructionActive := false
	AbleToMoveTimer := time.NewTimer(10 * time.Second)
	AbleToMoveTimerStarted := false
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
			if a.CurrentMasterID == MasterStruct.CurrentMasterID {
				MasterStruct = a
				MasterTimer.Stop()
				MasterTimer.Reset(MasterTimeout)
			}

		case <-MasterTimer.C:
			//Single elevator

			if len(Peerlist.Peers) == 0 { //must be zero
				HallRequests := MasterStruct.HallRequests
				if len(HallRequests) == 0 {
					HallRequests = [][2]bool{{false, false}, {false, false}, {false, false}, {false, false}}
				}
				CabRequests := MasterStruct.ElevStates[MyID].CabRequests
				if len(CabRequests) == 0 {
					CabRequests = [4]bool{false, false, false, false}
				}
				SingleElevRequests := requests.RequestsAppendHallCab(HallRequests, CabRequests)

				e := elevator.Elev{
					Behaviour:   elevator.EB_Idle,
					Floor:       elevio.GetFloor(),
					Dirn:        "stop",
					CabRequests: CabRequests,
				}
				if e.Floor == -1 {
					e = fsm.Fsm_onInitBetweenFloors(e)
				}
				for len(Peerlist.Peers) == 0 { //must be zero
					select {
					case a := <-drv_buttons:
						elevio.SetButtonLamp(a.Button, a.Floor, true)
						e, SingleElevRequests = fsm.Fsm_onRequestButtonPressed(e, SingleElevRequests, a.Floor, a.Button, doorTimer)

					case a := <-drv_floors:
						if a == numFloors-1 {
							e.Dirn = "stop"
						} else if a == 0 {
							e.Dirn = "stop"
						}
						e, SingleElevRequests = fsm.Fsm_onFloorArrival(e, SingleElevRequests, a, doorTimer)

					case a := <-drv_obstr:
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
							e, SingleElevRequests = fsm.Fsm_onDoorTimeout(e, SingleElevRequests)
						}
					case a := <-PeerUpdateCh:
						//what happens here
						Peerlist = a
						HallRequests, CabRequests = requests.RequestsSplitHallCab(SingleElevRequests)
						e.CabRequests = CabRequests
						MasterStruct.ElevStates[MyID] = e
						MasterStruct.HallRequests = HallRequests
						MasterStruct.CurrentMasterID = MyID
						err := exec.Command("gnome-terminal", "--", "go", "run", "../main.go", "master", "isolated").Run()
						fmt.Println(err)
						//Send masterstruct / mergestruct
					}
				}

			} else if Peerlist.Peers[0] == MyID { //I am master, start new master
				MasterStruct.CurrentMasterID = MyID
				err := exec.Command("gnome-terminal", "--", "go", "run", "../main.go", "master", "notIsolated").Run()
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

		case a := <-drv_buttons:
			buttonEvent := types.SlaveButtonEventMsg{
				ID:        MyID,
				Btn_floor: a.Floor,
				Btn_type:  int(a.Button)}
			slaveButtonTx <- buttonEvent

		case a := <-drv_floors:
			//Check for last floor in MasterStruct
			//Send can move if newfloor != masterstruct.floor
			//Stop timer
			fmt.Println("Current floor: ")
			fmt.Println(a)
			if a != MasterStruct.ElevStates[MyID].Floor {
				fmt.Println("Stopping timer, case floor")
				AbleToMoveTimer.Stop()
				AbleToMoveTimerStarted = false
				AbleToMoveCh <- types.AbleToMove{ID: MyID, AbleToMove: true}
			}
			floorEvent := types.SlaveFloor{ID: MyID, NewFloor: a}
			slaveFloorTx <- floorEvent
			elevio.SetFloorIndicator(a)
			elevio.SetMotorDirection(elevio.MD_Stop)

		case a := <-drv_obstr:
			//Send can't/can move
			if a {
				obstructionActive = true
				if doorIsOpen {
					AbleToMoveCh <- types.AbleToMove{ID: MyID, AbleToMove: false}
				}
			} else {
				obstructionActive = false
				AbleToMoveCh <- types.AbleToMove{ID: MyID, AbleToMove: true}
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

		case a := <-masterMotorDirRx:
			if a.ID == MyID && a.MasterID == MasterStruct.CurrentMasterID {

				//if "stop" stop timer

				if a.Motordir != "stop" && !AbleToMoveTimerStarted && !doorIsOpen {
					fmt.Println("UnableToMoveTimer started")
					AbleToMoveTimer.Stop()
					AbleToMoveTimer.Reset(3 * time.Second)
					AbleToMoveTimerStarted = true
				} else if a.Motordir == "stop" {
					AbleToMoveTimer.Stop()
					AbleToMoveTimerStarted = false
				}
				floor := elevio.GetFloor()
				if (floor == elevio.NumFloors-1 && a.Motordir == "up") || (floor == 0 && a.Motordir == "down") {
					floorEvent := types.SlaveFloor{ID: MyID, NewFloor: floor}
					slaveFloorTx <- floorEvent
				} else {
					if !doorIsOpen {
						elevio.SetMotorDirection(elevio.StringToMotorDir(a.Motordir))
					}
				}
			}
		case <-AbleToMoveTimer.C:
			fmt.Println()
			fmt.Println()
			fmt.Println()
			fmt.Println()
			fmt.Println("***********************************UnableToMove sent*****************************")
			fmt.Println()
			fmt.Println()
			fmt.Println()
			fmt.Println()

			AbleToMoveCh <- types.AbleToMove{ID: MyID, AbleToMove: false}
			AbleToMoveTimerStarted = false

		case a := <-masterSetOrderLight:
			if a.ID == MyID && a.MasterID == MasterStruct.CurrentMasterID {
				fsm.SetAllLights(a.LightOn)
			} else {
				fsm.SetOnlyHallLights(a.LightOn)
			}

		case a := <-commandDoorOpen:
			if a.MasterID == MasterStruct.CurrentMasterID {
				if a.ID == MyID && !doorIsOpen && elevio.GetFloor() != -1 {
					elevio.SetDoorOpenLamp(a.SetDoorOpen)
					elevio.SetMotorDirection(0)
					if a.SetDoorOpen {
						doorIsOpen = true
						AbleToMoveTimer.Stop()
						AbleToMoveTimerStarted = false
						doorTimer.Stop()
						doorTimer.Reset(3 * time.Second)
					}
					slaveDoorOpened <- a
				}
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
