package main

import (
	"fmt"
	"master/Driver-go/elevio"
	"master/elevator"
	"master/fsm"
	"master/network/broadcast"
	"master/network/localip"
	"master/network/peers"
	"master/requests"
	"master/types"
	"os/exec"
	"strconv"
	"time"
)

func main() {

	numFloors := 4
	elevio.Init("localhost:15657", numFloors)
	MyID, _ := localip.LocalIP()
	//MyID := "one"

	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)

	commandDoorOpenCh := make(chan types.DoorOpen)
	slaveButtonTx := make(chan types.SlaveButtonEventMsg)
	slaveFloorTx := make(chan types.SlaveFloor, 5)
	masterMotorDirRx := make(chan types.MasterCommand)
	masterSetOrderLightCh := make(chan types.SetOrderLight)
	slaveDoorOpenedCh := make(chan types.DoorOpen)
	transmitEnableCh := make(chan bool)
	masterMsgCh := make(chan types.MasterStruct)
	masterInitStructCh := make(chan types.MasterStruct)
	newMasterIDCh := make(chan types.NewMasterID)
	peerUpdateCh := make(chan peers.PeerUpdate)
	ableToMoveCh := make(chan types.AbleToMove)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	go broadcast.Transmitter(16513, slaveButtonTx)
	go broadcast.Transmitter(16514, slaveFloorTx)
	go broadcast.Transmitter(16521, slaveDoorOpenedCh)
	go broadcast.Transmitter(16527, masterInitStructCh)
	go broadcast.Transmitter(16528, ableToMoveCh)

	go broadcast.Receiver(16515, masterMotorDirRx)
	go broadcast.Receiver(16518, masterSetOrderLightCh)
	go broadcast.Receiver(16520, commandDoorOpenCh)
	go broadcast.Receiver(16523, masterMsgCh)
	go broadcast.Receiver(16524, newMasterIDCh)

	go peers.Transmitter(16529, MyID, transmitEnableCh)
	go peers.Receiver(16529, peerUpdateCh)

	var Peerlist peers.PeerUpdate
	InitTurnLightsOff := [4][3]bool{}
	fsm.SetAllLights(InitTurnLightsOff)
	elevio.SetDoorOpenLamp(false)
	floor := elevio.GetFloor()
	if floor == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
		floor = <-drv_floors
	}
	elevio.SetMotorDirection(elevio.MD_Stop)
	elevio.SetFloorIndicator(floor)

	exec.Command("gnome-terminal", "--", "go", "run", "../main.go", "init", MyID, "isolated", strconv.Itoa(floor)).Run()

	MySlaves := types.MySlaves{Active: []string{MyID}}
	MasterStruct := types.MasterStruct{
		CurrentMasterID: MyID,
		Isolated:        false,
		PeerList:        peers.PeerUpdate{},
		HallRequests:    [][2]bool{{false, false}, {false, false}, {false, false}, {false, false}},
		ElevStates:      map[string]elevator.Elev{},
		MySlaves:        MySlaves,
	}

	e := fsm.UnInitializedElev()
	e.Floor = floor
	MasterStruct.ElevStates[MyID] = e

	MasterTimeout := 5 * time.Second
	doorTimeout := 3 * time.Second
	AbleToMoveTimeout := 3 * time.Second

	doorTimer := time.NewTimer(doorTimeout)
	doorTimer.Stop()
	doorIsOpen := false
	obstructionActive := false

	AbleToMoveTimer := time.NewTimer(AbleToMoveTimeout)
	AbleToMoveTimer.Stop()
	AbleToMoveTimerStarted := false

	MasterTimer := time.NewTimer(MasterTimeout)

	for {
		select {
		case a := <-drv_buttons:
			buttonEvent := types.SlaveButtonEventMsg{
				ID:        MyID,
				Btn_floor: a.Floor,
				Btn_type:  int(a.Button)}
			slaveButtonTx <- buttonEvent

		case a := <-drv_floors:
			if a != MasterStruct.ElevStates[MyID].Floor {
				AbleToMoveTimer.Stop()
				AbleToMoveTimerStarted = false
				ableToMoveCh <- types.AbleToMove{ID: MyID, AbleToMove: true}
			}
			elevio.SetFloorIndicator(a)
			elevio.SetMotorDirection(elevio.MD_Stop)
			floorEvent := types.SlaveFloor{ID: MyID, NewFloor: a}
			slaveFloorTx <- floorEvent

		case a := <-drv_obstr:
			if a {
				obstructionActive = true
				ableToMoveCh <- types.AbleToMove{ID: MyID, AbleToMove: false}
			} else {
				obstructionActive = false
				ableToMoveCh <- types.AbleToMove{ID: MyID, AbleToMove: true}
				if doorIsOpen {
					doorTimer.Stop()
					doorTimer.Reset(3 * time.Second)
				}
			}

		case a := <-drv_stop:
			fmt.Printf("%+v\n", a)

		case a := <-masterMsgCh:
			if a.CurrentMasterID == MasterStruct.CurrentMasterID {
				MasterStruct = a
				MasterTimer.Stop()
				MasterTimer.Reset(MasterTimeout)
			}

		case a := <-newMasterIDCh:
			if a.SlaveID == MyID {
				MasterStruct.CurrentMasterID = a.NewMasterID
			}

		case NewPeerlist := <-peerUpdateCh:
			Peerlist = NewPeerlist

		case <-MasterTimer.C:
			if len(Peerlist.Peers) == 0 {
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
				fsm.SetAllLights(SingleElevRequests)
				if doorIsOpen && obstructionActive {
					elevio.SetDoorOpenLamp(true)
					e.Behaviour = elevator.EB_DoorOpen
				} else {
					elevio.SetDoorOpenLamp(false)
				}
				if e.Behaviour != elevator.EB_DoorOpen {
					NextAction := requests.RequestsNextAction(e, SingleElevRequests)
					fmt.Println(NextAction)
					if !obstructionActive {
						elevio.SetMotorDirection(NextAction.Dirn)
					}
					e.Behaviour = NextAction.Behaviour
					e.Dirn = elevio.MotorDirToString(NextAction.Dirn)
				} else {
					elevio.SetDoorOpenLamp(true)
					ClearHallReqs := requests.ShouldClearHallRequest(e, HallRequests)
					SingleElevRequests[e.Floor][0] = ClearHallReqs[0]
					SingleElevRequests[e.Floor][1] = ClearHallReqs[1]
					SingleElevRequests[e.Floor][2] = false
					fsm.SetAllLights(SingleElevRequests)
					e.Behaviour = elevator.EB_DoorOpen
					doorTimer.Stop()
					doorTimer.Reset(3 * time.Second)
				}
				for len(Peerlist.Peers) == 0 {
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
					case a := <-peerUpdateCh:
						Peerlist = a
						HallRequests, CabRequests = requests.RequestsSplitHallCab(SingleElevRequests)
						e.CabRequests = CabRequests
						MasterStruct.ElevStates[MyID] = e
						MasterStruct.HallRequests = HallRequests
						MasterStruct.CurrentMasterID = MyID
						if obstructionActive {
							ableToMoveCh <- types.AbleToMove{ID: MyID, AbleToMove: false}
						}
						exec.Command("gnome-terminal", "--", "go", "run", "../main.go", "master", MyID, "isolated").Run()
						MasterTimer.Stop()
						MasterTimer.Reset(MasterTimeout)
						IsolatedMasterStruct := MasterStruct
						go func() {
							for i := 0; i < 10; i++ {
								masterInitStructCh <- IsolatedMasterStruct
								time.Sleep(100 * time.Millisecond)
							}
						}()
					}
				}

			} else if Peerlist.Peers[0] == MyID {
				MasterStruct.CurrentMasterID = MyID
				exec.Command("gnome-terminal", "--", "go", "run", "../main.go", "master", MyID, "notIsolated").Run()
				MasterTimer.Stop()
				MasterTimer.Reset(MasterTimeout)

				go func(MasterStruct types.MasterStruct) {
					for i := 0; i < 10; i++ {
						masterInitStructCh <- MasterStruct
						time.Sleep(100 * time.Millisecond)
					}
				}(MasterStruct)
			} else {
				MasterTimer.Stop()
				MasterTimer.Reset(MasterTimeout)
			}

		case a := <-masterMotorDirRx:
			if a.ID == MyID && a.MasterID == MasterStruct.CurrentMasterID {
				if a.Motordir != "stop" && !AbleToMoveTimerStarted && !doorIsOpen {
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
			ableToMoveCh <- types.AbleToMove{ID: MyID, AbleToMove: false}
			AbleToMoveTimerStarted = false

		case a := <-masterSetOrderLightCh:
			if a.ID == MyID && a.MasterID == MasterStruct.CurrentMasterID {
				fsm.SetAllLights(a.LightOn)
			} else {
				fsm.SetOnlyHallLights(a.LightOn)
			}

		case a := <-commandDoorOpenCh:
			if a.MasterID == MasterStruct.CurrentMasterID {
				if a.ID == MyID && elevio.GetFloor() != -1 {
					if !doorIsOpen {
						elevio.SetDoorOpenLamp(a.SetDoorOpen)
						elevio.SetMotorDirection(0)
						if a.SetDoorOpen {
							if obstructionActive {
								ableToMoveCh <- types.AbleToMove{ID: MyID, AbleToMove: false}
								AbleToMoveTimer.Stop()
								AbleToMoveTimerStarted = false
							}
							doorIsOpen = true
							AbleToMoveTimer.Stop()
							AbleToMoveTimerStarted = false
							doorTimer.Stop()
							doorTimer.Reset(3 * time.Second)
						}
						slaveDoorOpenedCh <- a
					} else {
						slaveDoorOpenedCh <- a
					}
				}
			}

		case <-doorTimer.C:
			if !obstructionActive {
				doorIsOpen = false
				elevio.SetDoorOpenLamp(false)
				slaveDoorOpenedCh <- types.DoorOpen{ID: MyID, SetDoorOpen: false}
			} else {
				doorTimer.Stop()
				doorTimer.Reset(3 * time.Second)
			}
		}
	}
}
