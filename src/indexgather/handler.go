package indexgather

import (
	"acs/src/communication/sender"
	"acs/src/cryptolib"
	"acs/src/message"
	"acs/src/quorum"
	"acs/src/utils"
	"log"
)

var id int64
var iid int
var n int
var verbose bool
var members []int
var sleepTimerValue int

var valid utils.IntVec
var finalstatus utils.IntBoolMap
var setvalue utils.IntVec
var decidedvalue utils.IntVec

func UpdateList(input []int, v int) {
	valid.Insert(v, input)
}

func QueryStatus(v int) bool {
	return finalstatus.GetStatus(v)
}

func QueryValue(v int) []int {
	return decidedvalue.Get(v)
}

func Startindexgather(e int, v int, input []int) {
	// log.Printf("starting index gather epoch %v, instanceid %v, input %v", e, v, input)
	//p := fmt.Sprintf("starting index gather epoch %v, instanceid %v, input %v", e, v, input)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	msg := message.ReplicaMessage{
		Mtype:    message.IG_INFORM,
		Instance: v,
		Source:   id,
		TS:       utils.MakeTimestamp(),
		Payload:  utils.IntsToBytes(input),
		Epoch:    e,
	}

	msgbyte, err := msg.Serialize()
	if err != nil {
		log.Fatalf("failed to serialize IG message")
	}
	sender.MACBroadcast(msgbyte, message.IG)
}

func HandleIGMsg(inputMsg []byte) {
	tmp := message.DeserializeMessageWithSignature(inputMsg) // 解序列化
	input := tmp.Msg
	content := message.DeserializeReplicaMessage(input)
	mtype := content.Mtype

	if !cryptolib.VerifyMAC(content.Source, tmp.Msg, tmp.Sig) {
		log.Printf("[Authentication Error] The signature of ig message has not been verified.")
		return
	}

	// log.Printf("handling message from %v, type %v", source, mtype)
	switch mtype {
	case message.IG_INFORM:
		HandleInform(content)
	case message.IG_ACK:
		HandleAck(content)
	case message.IG_PREPARE:
		HandlePrepare(content)
	default:
		log.Printf("not supported")
	}
}

// func InitParametersForEpoch(e int, instanceids []int) {
//	epoch.Set(e)
//	allids = instanceids
// }

func Initindexgather(thisid int64, numNodes int, ver bool, mem []int, st int, v int) {
	id = thisid
	iid, _ = utils.Int64ToInt(id)
	n = numNodes
	verbose = ver
	quorum.StartQuorum(n)
	members = mem
	sleepTimerValue = st

	valid.Init()
	finalstatus.Init()
	setvalue.Init()
	decidedvalue.Init()
}
