package constmvba

import (
	"acs/src/aba/coin"
	aba "acs/src/aba/pisa"
	"acs/src/broadcast/wrbc"
	"acs/src/communication/sender"
	"acs/src/cryptolib"
	"acs/src/message"
	"acs/src/quorum"
	"acs/src/utils"
	"bytes"
	"log"
	"math/rand"
	"time"
)

type BAStatus int

const (
	STATUS_READY     BAStatus = 0
	STATUS_DECIDED   BAStatus = 1
	STATUS_TERMINATE BAStatus = 2
)

var baseinstance int
var finalstatus utils.IntIntMap // status for each instance, mape instance id to status
var round utils.IntIntMap
var curK utils.IntIntMap // tracking current instance number inside the consensus loop, map epoch to k
var astatus utils.IntBoolMap
var decidedvalue utils.IntVec  // map epoch to round number
var abastarted utils.BoolValue // used to track whether pi has already aba-proposed.
var permutation []int

func GetInstanceID(e int, r int) int {
	return r + n*e
}

func InitPer() {
	coin.InitCoin(n, id, quorum.SQuorumSize(), members, verbose)
	coin.SetMode(false) // obtain entire hash instead of one coin
}

func GetRandomPermutation(e int) []int { // shuffle of instance ids instead of node identities
	//p := fmt.Sprintf("MVBA [%v] start to get permutation", e)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	coinval := make(chan int)
	go GetCoin(e, 0, coinval)
	c := <-coinval

	shuffledids = allids
	rand.Seed(utils.IntToInt64(c))
	rand.Shuffle(len(shuffledids), func(i, j int) { shuffledids[i], shuffledids[j] = shuffledids[j], shuffledids[i] })
	return shuffledids
}

func GetCoin(instanceid int, roundnum int, r chan int) {

	waitTime := 1
	coin.ClearCoin(instanceid)
	coin.GenCoin(id, instanceid, roundnum)
	for {
		v := coin.QueryCoin(instanceid, roundnum)
		if v >= 0 {
			r <- v
			return
		}
		time.Sleep(time.Duration(waitTime) * time.Millisecond)
		waitTime = waitTime * 2
	}

}

func ProceedToIteration(e int) {
	rbc_t2 = utils.MakeTimestamp()
	permutation = GetRandomPermutation(e)

	//p := fmt.Sprintf("MVBA [%v] permutation  %v", e, permutation)
	//logging.PrintLog(verbose, logging.NormalLog, p)
	aba_t1 = utils.MakeTimestamp()
	StartLoop(e)
}

func StartLoop(e int) {
	abastarted.Set(false)
	r, _ := round.Get(e) // 获取当前epoch的轮数
	if r >= n {
		log.Printf("Faltal error, number of iterations is longer than n")
		return
	}

	k := permutation[r] // 获取k，election
	llock.Lock()
	curK.Insert(e, k)
	llock.Unlock()

	_, any := astatus.Get(k) // 检查RBC_k在astatus

	abainstanceid := GetInstanceID(e, r) // 获取aba的id
	if any {
		aba.StartABAFromRoundZero(abainstanceid, 1) // 已经提交，aba->1
	} else {
		abastarted.Set(true)
		aba.StartABAFromRoundZero(abainstanceid, 0) // 已经提交，aba->0
	}

	for { // 等待aba完成
		status := aba.QueryStatus(abainstanceid)
		if status {
			break
		} else {
			time.Sleep(time.Duration(2*sleepTimerValue) * time.Millisecond)
		}
	}

	value := aba.QueryValue(abainstanceid) // 获取aba结果

	if value == 1 { // aba结果为1
		aba_t2 = utils.MakeTimestamp()
		// if aba_t2-aba_t1 >= int64(sleepTimerValue) && rbc_t2-rbc_t1 >= int64(sleepTimerValue) && rbc_t3-rbc_t1 >= int64(sleepTimerValue) {
		// 	log.Printf("RBC1:RBC2:RABA %v %v %v", rbc_t3-rbc_t1, rbc_t2-rbc_t3, aba_t2-aba_t1)
		// 	p := fmt.Sprintf("RBC1:RBC2:RABA %v %v %v", rbc_t3-rbc_t1, rbc_t2-rbc_t3, aba_t2-aba_t1)
		// 	logging.PrintLog(true, logging.BreakdownLog, p)
		// }
		Decide(e, k, r) // 决定
	} else { // 为0
		round.Increment(e) // 轮数+1
		StartLoop(e)
	}
}

func GetRBCtT1(t1 int64) {
	rbc_t1 = t1
}

func Decide(e int, k int, r int) {
	for {
		h := wrbc.QueryReq(k)
		if h != nil {
			committedhash = h
			break
		} else {
			time.Sleep(time.Duration(2*sleepTimerValue) * time.Millisecond)
		}
	}
	vec := wrbc.QueryVal(k) // hash already checked

	if vec != nil {
		llock.Lock()
		finalstatus.Insert(e, int(STATUS_TERMINATE))
		decidedvalue.Insert(e, vec)
		llock.Unlock()

		msg := message.WRBCMessage{
			Mtype:    message.MVBA_DISTRIBUTE,
			Source:   id,
			Value:    vec,
			Instance: k,
			Epoch:    e,
		}

		msgbyte, err := msg.Serialize()
		if err != nil {
			log.Fatalf("failed to serialize RBC message")
		}
		sender.MACBroadcast(msgbyte, message.MVBA)
	}
	// todo: exchange message and wait for one from others
}

func RabaReprop(instanceid int, e int) {
	k, exi := curK.Get(e)
	if !exi {
		return
	}
	if instanceid != k {
		return
	}
	v := abastarted.Get()
	r, _ := round.Get(e)
	abainstanceid := GetInstanceID(e, r)
	if v {
		aba.StartABAFromRoundZero(abainstanceid, 1) // repropose
	}
}

func HandleDistributeMsg(inputMsg []byte) {
	tmpmsg := message.DeserializeMessageWithSignature(inputMsg)
	input := tmpmsg.Msg
	content := message.DeserializeWRBCMessage(input)

	llock.RLock()
	fs, exist := finalstatus.Get(content.Epoch)
	llock.RUnlock()
	if exist && fs == int(STATUS_TERMINATE) {
		return
	}

	if !cryptolib.VerifyMAC(content.Source, tmpmsg.Msg, tmpmsg.Sig) {
		log.Printf("[Authentication Error] The signature of distribute message has not been verified.")
		return
	}

	if content.Epoch != epoch.Get() {
		return
	}
	wmsg := message.WRBCMessage{
		Value: content.Value,
	}
	tmp, _ := wmsg.Serialize()
	curhash := cryptolib.GenInstanceHash(utils.IntToBytes(content.Instance), tmp)
	if bytes.Compare(committedhash, curhash) == 0 {
		e := content.Epoch
		llock.Lock()
		finalstatus.Insert(e, int(STATUS_TERMINATE))
		decidedvalue.Insert(e, content.Value)
		llock.Unlock()
	}
}
