package wrbc

import (
	"acs/src/communication/sender"
	"acs/src/cryptolib"
	"acs/src/message"
	"acs/src/quorum"
	"acs/src/utils"
	"log"
)

var id int64
var n int
var verbose bool
var epoch utils.IntValue
var sleepTimerValue int

func StartRBC(instanceid int, input []int) {
	//log.Printf("Starting RBC %v for epoch %v\n", instanceid, epoch.Get())
	//p := fmt.Sprintf("[%v] Starting RBC for epoch %v", instanceid, epoch.Get())
	//logging.PrintLog(verbose, logging.NormalLog, p)

	wrbcmsg := message.WRBCMessage{
		Mtype:    message.RBC_SEND,
		Instance: instanceid,
		Source:   id,
		Value:    input,
		Epoch:    epoch.Get(),
	}
	wrbcinput, err := wrbcmsg.Serialize()

	if err != nil {
		log.Fatalf("failed to serialize RBC message")
	}
	sender.MACBroadcast(wrbcinput, message.RBC)
}

func HandleRBCMsg(inputMsg []byte) {

	tmp := message.DeserializeMessageWithSignature(inputMsg)
	input := tmp.Msg
	content := message.DeserializeWRBCMessage(input)
	mtype := content.Mtype

	if !cryptolib.VerifyMAC(content.Source, tmp.Msg, tmp.Sig) {
		log.Printf("[Authentication Error] The signature of rbc message has not been verified.")
		return
	}

	//log.Printf("handling message from %v, type %v", source, mtype)
	switch mtype {
	case message.RBC_SEND:
		HandleSend(content)
	case message.RBC_ECHO:
		HandleEcho(content)
	case message.RBC_READY:
		HandleReady(content)
	default:
		log.Printf("not supported")
	}

}

func HandleCachedMsg(ep int) {

	msgs, any := cachedMsg.GetAndClear(ep)

	if !any {
		return
	}

	//p := fmt.Sprintf("[%v] Handling cached message, len(msgs) %v", ep, len(msgs))
	//logging.PrintLog(verbose, logging.NormalLog, p)

	log.Printf("=== Handle cache message")
	for i := 0; i < len(msgs); i++ {
		m := message.DeserializeWRBCMessage(msgs[i])
		switch m.Mtype {
		case message.RBC_SEND:
			HandleSend(m)
		case message.RBC_ECHO:
			HandleEcho(m)
		case message.RBC_READY:
			HandleReady(m)
		default:
			log.Printf("not supported")
		}
	}
}

func SetEpoch(e int) {
	epoch.Set(e)
}

func InitRBC(thisid int64, numNodes int, ver bool, st int) {
	id = thisid
	n = numNodes
	verbose = ver
	quorum.StartQuorum(n)
	//log.Printf("ini rstatus %v",rstatus.GetAll())
	rstatus.Init()
	instancestatus.Init()
	cachestatus.Init()
	receivedReq.Init()
	receivedList.Init()
	received.Init()
	epoch.Init()
	insList.Init()
	cachedMsg.Init()
	sleepTimerValue = st
}

func ClearRBCStatus(instanceid int) {
	rstatus.Delete(instanceid)
	instancestatus.Delete(instanceid)
	cachestatus.Delete(instanceid)
	receivedReq.Delete(instanceid)
}
