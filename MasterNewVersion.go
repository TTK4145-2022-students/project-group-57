//Ny versjon hvor slaven får ekstra ordre men ignorer ny ordre om den allerede har en. 


//har chan med id og bool 
MasterWatchTimers(elev.id, duration)
for 
a:= <- TimerSignal //blocker frem til vi får ett slikt signal 
if a.id == id 
	if (a.bool == 1){
		send timeout signal to right slave
	}else{
		stop then restart the correct timer. 
}

MasterMonitorElevs(channels?)
	starter timer. //kanskje en egen go funk? 
	for select
	case på alle mulige commandtyper. (Lys, dør, retning opp osv.)
	I hver av casene: Oppdaterer elevstates. restart timer. Sender signal om at vi har en ny tilstand.
	På en channel(Bortsett fra om det kun var alive msg) 
	Når vi får case door open kjører vi ClearRequestCurrentFloor
	og slår av lys. 
	
	Timeout om heisen ikke har sendt noe på ett visst antall sek. //kanskje egen go func
//kjøres som Go func

MasterGiveCommands
for
<- newStates //denne blocker frem til MasterMonitorElevs sender signal om at vi har ny tilstand. 
	HRAHallRequests = HRA()
	for each elev
	nextCommand = MasterRequestsNextAction(elev, this elevs HRAHallRequests)
	send next command

