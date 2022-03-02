package timer

import "time"

func TimerStart() {
	timer1 := time.NewTimer(3 * time.Second)

	<-timer1.C
}
