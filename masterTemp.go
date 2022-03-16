//Sjekker etter nye states/requests
//kjører HRA igjen. 
//får ut liste over orders til alle 3 elevators. 
//kanskje disse ovenfor skjer i main, og master får tilsendt listene med orders og elevator states.



//Før denne kjøres sjekker vi om flagget CommandActiveFlag er aktiv
//Denne kjøres kun dersom vi ikke allerede kjører en command. Med denne løsningen må vi altså legge
//til et slikt flag i det Master lagrer om heisene. 
func MasterGiveCommand(tar inn heis id, og en command)
	UDP broadcaster commanden til heisen med rett ID. 
}//etter denne har kjørt (kanskje ett visst antall ganger?) settes flagget til true. 


//kjøres som en goRoutine. 
func MasterWatchCommand(heis id + state, og samme command){
	starter timer. 
	switch på alle mulige commandtyper. (Lys, dør, retning opp osv.)
	I hver av casene: Lytter på denne rette kanalen
	Dersom heisen har utført orderen korrekt. Feks ankommet 3 etasje.
	Send signal om at orderen ble fullført. 
	og avslutt funksjonen

	Dersom heisen har failet commanden. Feks ankommet 3 når den skulle til 1. Send Timeout signal.

	Dersom timeren går ut før heisen har fullført orderen. Send timeout signal. 
	//om vi vil optimalisere kan vi ha forskjellige timeout tider på forskjellige ordere. 
}
//Dersom denne funksjonen sender signal om at orderen ble fullført må flagget settes til false.


//slik ser jeg for meg disse funksjonene blir kalt.

chan CommandDone (Denne channelen har elev.id og en bool som input)
for{
	HRAHallRequests = HRA()
	for hver elev{
		if elev.CommandActiveFlag == true{
			do nothing
		}else{
			nextCommand = MasterRequestsNextAction(elev, HRAHallRequests)
			MasterGiveCommand(elev.id, nextCommand)
			elev.CommandActiveFlag == true
			go MasterWatchCommand(elev.id, elev.state, NextCommand{elev}) 
		}
	}
	WatchCommandFinished := <- CommandDone
	if WatchCommandFinished.flag == 1{//heisen fullførte orderen korrekt
		Her setter vi CommandActiveFlag for heisen med WatchCommandFinished.id til false. 
	}else{
		heisen timouta og restartes. 
	}
}

