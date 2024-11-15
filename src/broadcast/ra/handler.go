package ra

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
var wrbc bool

func StartRA(instanceid int, input []byte) {
	//log.Printf("Starting RBC %v for epoch %v\n", instanceid, epoch.Get())
	//p := fmt.Sprintf("[%v] Starting RBC for epoch %v", instanceid, epoch.Get())
	//logging.PrintLog(verbose, logging.NormalLog, p)

	msg := message.ReplicaMessage{
		Mtype:    message.RA_ECHO,
		Instance: instanceid,
		Source:   id,
		TS:       utils.MakeTimestamp(),
		Payload:  input,
		Epoch:    epoch.Get(),
	}

	msgbyte, err := msg.Serialize()
	if err != nil {
		log.Fatalf("failed to serialize RBC message")
	}
	sender.MACBroadcast(msgbyte, message.RA)
}

func HandleRAMsg(inputMsg []byte) {

	tmp := message.DeserializeMessageWithSignature(inputMsg) //解序列化
	input := tmp.Msg
	content := message.DeserializeReplicaMessage(input)
	mtype := content.Mtype

	if !cryptolib.VerifyMAC(content.Source, tmp.Msg, tmp.Sig) {
		log.Printf("[Authentication Error] The signature of rbc message has not been verified.")
		return
	}

	//log.Printf("handling message from %v, type %v", source, mtype)
	switch mtype {
	case message.RA_ECHO:
		HandleEcho(content)
	case message.RA_READY:
		HandleReady(content)
	default:
		log.Printf("not supported")
	}

}

func SetHash() { // use hash for the echo and ready phase
	wrbc = true
}

func SetEpoch(e int) {
	epoch.Set(e)
}

func InitRA(thisid int64, numNodes int, ver bool) {
	id = thisid
	n = numNodes
	verbose = ver
	quorum.StartQuorum(n)
	wrbc = false
	//log.Printf("ini rstatus %v",rstatus.GetAll())
	rstatus.Init()
	instancestatus.Init()
	cachestatus.Init()
	receivedReq.Init()
	received.Init()
	receivedSig.Init()
	epoch.Init()
}

func ClearRBCStatus(instanceid int) {
	rstatus.Delete(instanceid)
	instancestatus.Delete(instanceid)
	cachestatus.Delete(instanceid)
	receivedReq.Delete(instanceid)
}
