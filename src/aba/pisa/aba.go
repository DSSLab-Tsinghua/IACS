package cobalt

import (
	"acs/src/aba/coin"
	"acs/src/communication/sender"
	"acs/src/config"
	"acs/src/message"
	"acs/src/quorum"
	"acs/src/utils"
	"log"
	"sync"
	"time"
)

type ABAStatus int

const (
	STATUS_READY     ABAStatus = 0
	STATUS_AUX       ABAStatus = 1
	STATUS_DECIDED   ABAStatus = 2
	STATUS_TERMINATE ABAStatus = 3
)

var baseinstance int
var round utils.IntIntMap //round number
var maj utils.IntIntMapArr
var delta utils.IntIntMapArr //\delta_r
var coinr utils.IntIntMapArr //\delta_r
var majs utils.IntIntSetMap  //majs values

var bvals utils.IntIntSetMap //set of sent bval values
var bin_values utils.IntIntSetMap
var auxvals utils.IntIntSetMap
var auxnodes utils.IntIntInt64SetMap
var auxmajvals utils.IntIntSetMap

var instancestatus utils.IntIntMap // status for each instance
var finalstatus utils.IntIntMap    // status for each instance
var decidedround utils.IntIntMap
var decidedvalue utils.IntIntMap
var alock sync.Mutex
var block sync.Mutex
var clock sync.Mutex
var dlock sync.Mutex
var cachedMsg utils.IntIntBytesMapArr
var astatus utils.IntSetMap
var bvalMap utils.IntIntDoubleSetMap
var abasetvalue utils.IntByteMap

func HandleCachedMsg(instanceid int, r int) {
	ro, _ := round.Get(instanceid) //是否为本轮
	if ro != r {
		return
	}
	stat, _ := finalstatus.Get(instanceid)
	if stat == int(STATUS_TERMINATE) { //是否已经结束
		return
	}

	msgs := cachedMsg.GetAndClear(instanceid, r)

	if len(msgs) == 0 {
		return
	}

	//p := fmt.Sprintf("[%v] Handling cached message for round %v, len(msgs) %v", instanceid, r, len(msgs))
	//logging.PrintLog(verbose, logging.NormalLog, p)

	for i := 0; i < len(msgs); i++ {
		m := message.DeserializeReplicaMessage(msgs[i])
		switch m.Mtype {
		case message.ABA_BVAL:
			go HandleBVAL(m)
		case message.ABA_AUX:
			go HandleAUX(m)
		}
	}

}

func CacheMsg(instanceid int, roundnum int, msg []byte) {
	cachedMsg.InsertValue(instanceid, roundnum, msg)
}

func HandleBVAL(m message.ReplicaMessage) {
	//p := fmt.Sprintf("[%v] Handling bval message, round %v, vote %v, maj %v from %v", m.Instance, m.Round, m.Value, m.Maj, m.Source)
	//logging.PrintLog(verbose, logging.NormalLog, p)
	r, _ := round.Get(m.Instance)

	if m.Round < r {
		return
	}
	if m.Round > r {
		msgbyte, _ := m.Serialize()
		CacheMsg(m.Instance, m.Round, msgbyte)
		return
	}

	HandleCachedMsg(m.Instance, r)

	bvalMap.Insert(m.Instance, m.Round, m.Value, m.Source)
	majs.InsertValue(m.Instance, m.Round, m.Maj)

	if bvalMap.GetCount(m.Instance, m.Round, m.Value) >= quorum.SQuorumSize() {
		SendBval(m.Instance, m.Round, m.Value)
		if (r == 0) && (m.Value == 1) {
			bin_values.InsertValue(m.Instance, m.Round, m.Value)
		}
		if changevote && m.Value == 1 && !bin_values.Contains(m.Instance, 0, m.Value) {
			log.Printf("change vote")
			StartABAFromRoundZero(m.Instance, 1)
		}
	}

	if bvalMap.GetCount(m.Instance, m.Round, m.Value) >= quorum.QuorumSize() {
		ProceedToAux(m)
	}

}

func HandleAUX(m message.ReplicaMessage) {
	r, _ := round.Get(m.Instance)

	//p := fmt.Sprintf("[%v] handling aux message, m.Round %v, round %v, vote %v, maj %v, from node %v", m.Instance, m.Round, r, m.Value, m.Maj, m.Source)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	if m.Round < r {
		//log.Printf("[%v] lower round %v %v",m.Instance,m.Round,r)
		return
	}
	if m.Round > r {
		//log.Printf("[%v] greater round %v %v",m.Instance,m.Round,r)
		msgbyte, _ := m.Serialize()
		CacheMsg(m.Instance, m.Round, msgbyte)
		return
	}

	go HandleCachedMsg(m.Instance, r)

	if m.Maj != 2 && !bin_values.Contains(m.Instance, m.Round, m.Maj) {
		//log.Printf("--------[%v] m.Maj %v, binv %v, source %v", m.Instance, m.Maj, bin_values.Get(m.Instance),m.Source)
		//p := fmt.Sprintf("[%v] round %v, m.Maj %v, binv %v, source %v", m.Instance, m.Round, m.Maj, bin_values.GetValue(m.Instance, m.Round), m.Source)
		//logging.PrintLog(verbose, logging.NormalLog, p)
		msgbyte, _ := m.Serialize()
		CacheMsg(m.Instance, m.Round, msgbyte)
		return
	}
	if m.Value != 2 && !bin_values.Contains(m.Instance, m.Round, m.Value) { //|| (m.Value==2 && (!bin_values.Contains(m.Instance,m.Round,0) || !bin_values.Contains(m.Instance,m.Round,1))){
		msgbyte, _ := m.Serialize()
		//p := fmt.Sprintf("[%v] round %v, m.Value %v, binv %v, source %v", m.Instance, m.Round, m.Value, bin_values.GetValue(m.Instance, m.Round), m.Source)
		//logging.PrintLog(verbose, logging.NormalLog, p)
		//log.Printf("--------[%v] m.Value %v, binv %v, source %v", m.Instance, m.Value, bin_values.Get(m.Instance),m.Source)
		CacheMsg(m.Instance, m.Round, msgbyte)
		return
	}
	var exist = true
	deltav := delta.GetValue(m.Instance, m.Round)
	if deltav == -1 {
		exist = false
	}
	if r > 0 {
		if m.Value == 1-m.Maj || m.Maj == 2 || (exist && deltav != m.Value) {
			log.Printf("[%v] round %v, invalid value %v, maj %v, deltav %v from %v", m.Instance, r, m.Value, m.Maj, deltav, m.Source)
			return
		}
	}

	clock.Lock()
	defer clock.Unlock()

	auxvals.InsertValue(m.Instance, m.Round, m.Value)
	auxmajvals.InsertValue(m.Instance, m.Round, m.Maj)
	//log.Printf("[%v] insert aux node %v, round %v",m.Instance,m.Source,m.Round)
	auxnodes.Insert(m.Instance, m.Round, m.Source)

	dr, _ := decidedround.Get(m.Instance)
	stat, _ := finalstatus.Get(m.Instance)
	rstat, _ := instancestatus.Get(m.Instance)
	if rstat == int(STATUS_AUX) && dr != r && stat == int(STATUS_DECIDED) {
		finalstatus.Insert(m.Instance, int(STATUS_TERMINATE))
		//p := fmt.Sprintf("[%v] terminate in round %v", m.Instance, r)
		//logging.PrintLog(verbose, logging.NormalLog, p)

		//clock.Unlock()
		return
	}
	if stat == int(STATUS_TERMINATE) {
		//clock.Unlock()
		return
	}
	var estr int
	var majr int

	r2, _ := round.Get(m.Instance)
	if r2 != r {
		//log.Printf("[%v] different round aux msg %v %v.",m.Instance,r2,r)
		//clock.Unlock()
		return
	}
	if auxnodes.GetLen(m.Instance, m.Round) >= quorum.QuorumSize() {
		//log.Printf("[%v] start get coin for round %v", m.Instance, r)
		//p := fmt.Sprintf("[%v] start get coin for round %v", m.Instance, r)
		//logging.PrintLog(verbose, logging.NormalLog, p)
		coinval := make(chan int)
		go GetCoin(m.Instance, r, coinval)
		c := <-coinval
		//log.Printf("[%v] coin for round %v, %v", m.Instance, r, c)
		//p = fmt.Sprintf("[%v] coin for round %v, %v", m.Instance, r, c)
		//logging.PrintLog(verbose, logging.NormalLog, p)
		var firstauxval int
		tmpa := auxvals.GetValue(m.Instance, m.Round)
		if len(tmpa) == 0 {
			return
		} else {
			firstauxval = tmpa[0]
		}
		if auxvals.GetCount(m.Instance, m.Round, 1) >= quorum.QuorumSize() {
			//log.Printf("[%v] get a quorum of 1 in round %v", m.Instance, r)
			estr = 1
			majr = 1
			if estr == c {
				Decide(m.Instance, r, estr)
			}
		} else if auxvals.GetCount(m.Instance, m.Round, 0) >= quorum.QuorumSize() {
			//log.Printf("[%v] get a quorum of 0 in round %v", m.Instance, r)
			estr = 0
			majr = 0
			if estr == c {
				Decide(m.Instance, r, estr)
			}
		} else if r > 0 && ((auxvals.GetLen(m.Instance, m.Round) == 1 && firstauxval == 2) || (auxvals.GetLen(m.Instance, m.Round) == 1 && auxvals.GetCount(m.Instance, m.Round, firstauxval) < quorum.QuorumSize())) {
			//log.Printf("[%v] enter second condition in round %v", m.Instance, r)
			//firstauxmajval := auxmajvals.GetValue(m.Instance,m.Round)[0]
			cr := coinr.GetValue(m.Instance, m.Round-1)
			/*if firstauxval == 2 && auxmajvals.GetLen(m.Instance,m.Round) == 1 && auxmajvals.GetCount(m.Instance,m.Round,firstauxmajval) >= quorum.QuorumSize(){
				estr = firstauxmajval
				majr = firstauxmajval
				if estr == cr && estr == c{
					Decide(m.Instance,r,estr)
				}
			}else*/
			if firstauxval != 2 && auxmajvals.GetCount(m.Instance, m.Round, firstauxval) >= quorum.QuorumSize() {
				estr = firstauxval
				majr = firstauxval
				if estr == cr && estr == c {
					Decide(m.Instance, r, estr)
				}
			} else if auxmajvals.GetLen(m.Instance, m.Round) == 2 {
				estr = cr
				majr = cr
			} else {
				estr, majr = UseCoinVal(m.Instance, c, r)
			}

		} else {
			//log.Printf("[%v] using common coin %v to enter round %v", m.Instance, c, r+1)
			estr, majr = UseCoinVal(m.Instance, c, r)
		}
		//log.Printf("[%v] enter the next round %v with %v, majr %v", m.Instance,r+1, estr, majr)
		//p = fmt.Sprintf("[%v] enter the next round %v with %v, majr %v", m.Instance, r+1, estr, majr)
		//logging.PrintLog(verbose, logging.NormalLog, p)

		dlock.Lock()
		defer dlock.Unlock()
		r2, _ = round.Get(m.Instance)
		if r2 != r {
			return
		}
		rstat, _ := instancestatus.Get(m.Instance)

		if rstat == int(STATUS_AUX) {

			InitParametersForInstance(m.Instance, r)
			//round.Increment(m.Instance)
			round.Set(m.Instance, r+1)
			maj.InsertValue(m.Instance, r+1, majr)
			coinr.InsertValue(m.Instance, r, c)

			StartABA(m.Instance, estr)
			instancestatus.Insert(m.Instance, int(STATUS_READY))
		}

	}
	//clock.Unlock()

}

func UseCoinVal(instanceid int, c int, r int) (int, int) {
	var estr int
	var majr int
	estr = c
	if r == 0 {
		majr = c
	} else {
		if auxvals.GetCount(instanceid, r, 0) > quorum.QuorumSize()/2 {
			majr = 0
		} else if auxvals.GetCount(instanceid, r, 1) > quorum.QuorumSize()/2 {
			majr = 1
		} else {
			majr = 2
		}
	}
	return estr, majr
}

func Decide(instanceid int, r int, estr int) {
	finalstatus.Insert(instanceid, int(STATUS_DECIDED))
	decidedround.Insert(instanceid, r)
	decidedvalue.Insert(instanceid, estr)
	//log.Printf("[%v] decide %v in round %v", instanceid, estr, r)
	//p := fmt.Sprintf("[%v] decide %v in round %v", instanceid, estr, r)
	//logging.PrintLog(verbose, logging.NormalLog, p)
}

func SendBval(instanceid int, roundnum int, value int) {

	//r,_ := round.Get(instanceid)
	if config.MaliciousNode() {
		switch config.MaliciousMode() {
		case 2:
			intid, err := utils.Int64ToInt(id)
			if err != nil {
				log.Fatal("Failed transform int64 to int", err)
			}
			//if intid > 2 * quorum.FSize(){
			if intid < quorum.FSize() {
				log.Printf("[%v] I'm a malicious node %v, send bval %v!", instanceid, id, 0)
				value = 0
			}
		case 3:
			intid, err := utils.Int64ToInt(id)
			if err != nil {
				log.Fatal("Failed transform int64 to int", err)
			}
			if intid < quorum.FSize() {
				if value != 2 {
					log.Printf("[%v] I'm a malicious node %v, send bval %v but not %v!", instanceid, id, value^1, value)
					value = value ^ 1
				}
			}
		}
	}

	if bvals.Contains(instanceid, roundnum, value) {
		return
	}

	bvals.InsertValue(instanceid, roundnum, value)
	//p := fmt.Sprintf("[%v] Sending bval %v since it previously has not", instanceid, value)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	mval := maj.GetValue(instanceid, roundnum)
	msg := message.ReplicaMessage{
		Mtype:    message.ABA_BVAL,
		Instance: instanceid,
		Source:   id,
		Value:    value,
		Maj:      mval,
		Round:    roundnum,
	}

	msgbyte, err := msg.Serialize()
	if err != nil {
		log.Fatalf("failed to serialize bval ABA message")
	}
	sender.MACBroadcast(msgbyte, message.ABA)
}

func ProceedToAux(m message.ReplicaMessage) {
	bin_values.InsertValue(m.Instance, m.Round, m.Value)
	//log.Printf("[%v] bin_values inseet %v::%v",m.Instance,m.Value,bin_values.Get(m.Instance))
	alock.Lock()
	r, _ := round.Get(m.Instance)
	if r != m.Round {
		alock.Unlock()
		return
	}
	stat, _ := instancestatus.Get(m.Instance)
	if stat >= int(STATUS_AUX) {
		alock.Unlock()
		return
	}
	instancestatus.Insert(m.Instance, int(STATUS_AUX))
	alock.Unlock()

	//coinvalr,_ := coinr.Get(m.Instance)

	proposedval := 2
	if r == 0 {
		delta.InsertValue(m.Instance, m.Round, m.Value)
		proposedval = m.Value
	}
	if r > 0 {
		coinvalr := coinr.GetValue(m.Instance, r-1)
		if (m.Value == 1-coinvalr && !majs.Contains(m.Instance, r, coinvalr) && !majs.Contains(m.Instance, r, 2)) || (m.Value == coinvalr && !majs.Contains(m.Instance, r, 1-coinvalr)) {
			delta.InsertValue(m.Instance, m.Round, m.Value)
			proposedval = m.Value
		}
	}
	//p := fmt.Sprintf("[%v] check %v %v %v %v %v",m.Instance,r,r==0,m.Value == 1-coinvalr && !majs.Contains(m.Instance,coinvalr) && !majs.Contains(m.Instance,2),m.Value==coinvalr && !majs.Contains(m.Instance,1-coinvalr),majs)
	//logging.PrintLog(verbose, logging.NormalLog, p)
	//if r==0 || ((m.Value == 1-coinvalr && !majs.Contains(m.Instance,coinvalr) && !majs.Contains(m.Instance,2)) || (m.Value==coinvalr && !majs.Contains(m.Instance,1-coinvalr))){

	//delta.InsertValue(m.Instance,m.Round, m.Value)
	//proposedval = m.Value
	//}

	//log.Printf("[%v] Sending aux %v, maj %v for round %v", m.Instance, proposedval, m.Value, r)
	//p := fmt.Sprintf("[%v] Sending aux %v, maj %v for round %v", m.Instance, proposedval, m.Maj, r)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	msg := message.ReplicaMessage{
		Mtype:    message.ABA_AUX,
		Instance: m.Instance,
		Source:   id,
		Value:    proposedval,
		Maj:      m.Value,
		Round:    r,
	}

	if config.MaliciousNode() {
		switch config.MaliciousMode() {
		case 2:
			intid, err := utils.Int64ToInt(id)
			if err != nil {
				log.Fatal("Failed transform int64 to int", err)
			}
			//if intid > 2 * quorum.FSize(){
			if intid < quorum.FSize() {
				log.Printf("[%v] I'm a malicious node %v, send aux %v for round %v!", m.Instance, id, 0, r)
				msg.Value = 0
			}
		case 3:
			intid, err := utils.Int64ToInt(id)
			if err != nil {
				log.Fatal("Failed transform int64 to int", err)
			}
			if intid < quorum.FSize() {
				if msg.Value != 2 {
					log.Printf("[%v] I'm a malicious node %v, send aux %v but not %v for round %v!", m.Instance, id, msg.Value^1, msg.Value, r)
					msg.Value = msg.Value ^ 1
				}
			}
		}
	}

	msgbyte, err := msg.Serialize()
	if err != nil {
		log.Fatalf("failed to serialize aux ABA message")
	}
	sender.MACBroadcast(msgbyte, message.ABA)
}

func GetCoin(instanceid int, roundnum int, r chan int) {
	if cointype && roundnum == 0 {
		r <- 1
		return
	}
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
