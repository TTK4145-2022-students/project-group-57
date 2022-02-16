package orders

import (
	"Driver-go/elevio"
	"elevio"
)

type Elevator struct {
	floor    int
	dirn     elevio.MotorDirection
	requests [elevio._numFloors][elevio.ButtonType]int
}
