package cobalt

import (
	"acs/src/aba/coin"
	"acs/src/communication/sender"
	"acs/src/config"
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
var mapMembers map[int]int
var cointype bool
var changevote bool

func QueryStatus(instanceid int) bool {
	v, exist := finalstatus.Get(instanceid)
	return exist && v >= int(STATUS_DECIDED)
}

func QueryValue(instanceid int) int {
	v, exist := decidedvalue.Get(instanceid)
	if !exist {
		return -1
	}
	return v
}

// TODO: this is not a full VBA yet. Need to verify signature if local value is not set yet.
func QueryHash(instanceid int) []byte {
	v, exist := abasetvalue.Get(instanceid)
	if !exist {
		return nil
	}
	return v
}

func SetValue(instanceid int, input []byte) {
	abasetvalue.Insert(instanceid, input)
}

func StartABAFromRoundZero(instanceid int, input int) { // Pisa: propose and repropose

	r := 0

	//p := fmt.Sprintf("[%v] Starting ABA round %v with value %v", instanceid, r, input)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	HandleCachedMsg(instanceid, r)

	if config.MaliciousNode() { //拜占庭节点的操作
		switch config.MaliciousMode() {
		case 2: //只发0
			intid, err := utils.Int64ToInt(id)
			if err != nil {
				log.Fatal("Failed transform int64 to int", err)
			}
			//if intid > 2 * quorum.FSize(){		//(2f+1)--(3f) are the malicious nodes that always vote 0 in ABA
			if intid < quorum.FSize() { //0--(f-1) are the malicious modes that always vote 0 in ABA
				log.Printf("[%v] I'm a malicious node %v, start ABA FromRoundZero %v for round %v!", instanceid, id, 0, r)
				input = 0
			}
		case 3: //发input ^ 1
			intid, err := utils.Int64ToInt(id)
			if err != nil {
				log.Fatal("Failed transform int64 to int", err)
			}
			if intid < quorum.FSize() {
				if input != 2 {
					log.Printf("[%v] I'm a malicious node %v, start ABA FromRoundZero %v but not %v for round %v!", instanceid, id, input^1, input, r)
					input = input ^ 1
				}
			}
		}
	}

	bvals.InsertValue(instanceid, r, input)
	maj.InsertValue(instanceid, r, 2)

	msg := message.ReplicaMessage{
		Mtype:    message.ABA_BVAL,
		Instance: instanceid,
		Source:   id,
		Value:    input,
		Maj:      2, //2 in round 0
		Round:    r,
	}
	v, exist := abasetvalue.Get(instanceid)
	if exist {
		msg.Payload = v
	}

	msgbyte, err := msg.Serialize()
	if err != nil {
		log.Fatalf("failed to serialize ABA message")
	}
	sender.MACBroadcast(msgbyte, message.ABA)

	if input == 1 {
		bin_values.InsertValue(instanceid, r, 1)
		//log.Printf("[%v] bin_values inseet 1 bias::%v",instanceid,bin_values.Get(instanceid))
		ProceedToAux(msg)
	}

}

func StartABA(instanceid int, input int) {

	r, _ := round.Get(instanceid)

	//p := fmt.Sprintf("[%v] Starting ABA round %v with value %v", instanceid, r, input)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	HandleCachedMsg(instanceid, r)

	if config.MaliciousNode() {
		switch config.MaliciousMode() {
		case 2:
			intid, err := utils.Int64ToInt(id)
			if err != nil {
				log.Fatal("Failed transform int64 to int", err)
			}
			//if intid > 2 * quorum.FSize(){
			if intid < quorum.FSize() {
				log.Printf("[%v] I'm a malicious node %v, start ABA %v for round %v!", instanceid, id, 0, r)
				input = 0
			}
		case 3:
			intid, err := utils.Int64ToInt(id)
			if err != nil {
				log.Fatal("Failed transform int64 to int", err)
			}
			if intid < quorum.FSize() {
				log.Printf("[%v] I'm a malicious node %v, start ABA %v for round %v!", instanceid, id, input^1, r)
				if input != 2 {
					input = input ^ 1
				}
			}
		}
	}

	bvals.InsertValue(instanceid, r, input)

	msg := message.ReplicaMessage{
		Mtype:    message.ABA_BVAL,
		Instance: instanceid,
		Source:   id,
		Value:    input,
		Round:    r,
	}

	if r == 0 {
		maj.InsertValue(instanceid, r, 2)
		msg.Maj = 2 //\bot for round 0
	} else {
		mval := maj.GetValue(instanceid, r)
		msg.Maj = mval
	}

	msgbyte, err := msg.Serialize()
	if err != nil {
		log.Fatalf("failed to serialize ABA message")
	}
	sender.MACBroadcast(msgbyte, message.ABA)

}

func HandleABAMsg(inputMsg []byte) {

	tmp := message.DeserializeMessageWithSignature(inputMsg)
	input := tmp.Msg
	content := message.DeserializeReplicaMessage(input)
	mtype := content.Mtype

	if !cryptolib.VerifyMAC(content.Source, tmp.Msg, tmp.Sig) {
		log.Printf("[Authentication Error] The signature of aba message has not been verified.")
		return
	}

	//log.Printf("handling message from %v, type %v", source, mtype)
	switch mtype {
	case message.ABA_BVAL:
		HandleBVAL(content)
	case message.ABA_AUX:
		HandleAUX(content)
	default:
		log.Printf("not supported")
	}
}

func InitCoinType(ct bool) {
	cointype = ct
}

func InitABA(thisid int64, numNodes int, ver bool, mem []int, st int) {
	id = thisid
	iid, _ = utils.Int64ToInt(id)
	n = numNodes
	verbose = ver
	quorum.StartQuorum(n)
	members = mem
	sleepTimerValue = st

	round.Init()

	//initialize round numbers to 0 for all instances
	mapMembers = make(map[int]int)
	for i := 0; i < len(members); i++ {
		round.Insert(members[i], 0)
		mapMembers[members[i]] = i
	}

	InitParameters()

	instancestatus.Init()
	finalstatus.Init()
	decidedround.Init()
	decidedvalue.Init()
	abasetvalue.Init()

	astatus.Init()
	baseinstance = 1000 //hard-code to 1000 to avoid conflicts

	coin.InitCoin(n, id, quorum.SQuorumSize(), mem, ver)
}

func SetChangeVoteMode() {
	changevote = true
}

func InitParameters() {
	cachedMsg.Init(n)
	bvals.Init()
	bin_values.Init()
	auxvals.Init()
	auxnodes.Init()
	auxmajvals.Init()
	bvalMap.Init()
	maj.Init()
	delta.Init()
	coinr.Init()
	majs.Init()
}

func InitParametersForInstance(instanceid int, r int) {
	astatus.Delete(instanceid)
}

func GetIndex(instanceid int) int {
	return mapMembers[instanceid]
}
