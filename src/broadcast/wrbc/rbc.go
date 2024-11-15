package wrbc

import (
	"acs/src/communication/sender"
	"acs/src/cryptolib"
	"acs/src/message"
	"acs/src/quorum"
	"acs/src/utils"
	"bytes"
	"log"
	"sync"
	"time"
)

type RBCStatus int

const (
	STATUS_IDLE  RBCStatus = 0
	STATUS_SEND  RBCStatus = 1
	STATUS_ECHO  RBCStatus = 2
	STATUS_READY RBCStatus = 3
)

var rstatus utils.IntBoolMap       //broadcast status,only has value when  RBC Deliver
var instancestatus utils.IntIntMap // status for each instance, used in RBC
var cachestatus utils.IntIntMap    // status for each instance
var receivedReq utils.IntByteMap   //store the hash of the vector
var receivedList utils.IntVec      //store the received vector from P_s
var received utils.IntSet
var elock sync.Mutex
var rlock sync.Mutex
var llock sync.RWMutex
var ulock sync.RWMutex
var insList utils.IntBoolMap
var cachedMsg utils.IntBytesMap

// check whether the instance has been deliver in RBC
func QueryStatus(instanceid int) bool {
	v, exist := rstatus.Get(instanceid)
	return v && exist
}

func QueryStatusCount() int {
	return rstatus.GetCount()
}

func QueryReq(instanceid int) []byte {
	v, exist := receivedReq.Get(instanceid)
	if !exist {
		return nil
	}
	return v
}

func QueryVal(instanceid int) []int {
	return receivedList.Get(instanceid)
}

func UpdateList(input []int) { // used for external predicate
	ulock.Lock()
	defer ulock.Unlock()
	//log.Printf("-------updated list %v", input)
	for i := 0; i < len(input); i++ {
		insList.Insert(input[i], true)
	}
}

func CheckPredicate(instanceid int) bool {
	llock.Lock()
	defer llock.Unlock()
	cs, _ := insList.Get(instanceid) // external predicate
	if !cs {
		return false
	}
	return true
}

func IterateCachedMsg(ep int) {
	for {
		e := epoch.Get()
		if e != ep {
			return
		}
		HandleCachedMsg(ep) // use instance as round number
		time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
	}

}

func HandleSend(m message.WRBCMessage) {
	result, exist := rstatus.Get(m.Instance)
	if exist && result {
		return
	}

	if epoch.Get() != m.Epoch {
		log.Printf("epoch %v, mepoch %v", epoch.Get(), m.Epoch)
		return
	}

	if !CheckPredicate(m.Instance) { //谓词不满足
		log.Printf("predicate not satisfied yet for ins %v", m.Instance)
		//p := fmt.Sprintf("[%v] Predicate not satisfied yet. Cache the message from node %v", m.Instance, m.Source)
		//logging.PrintLog(verbose, logging.NormalLog, p)

		msgbyte, _ := m.Serialize()
		CacheMsg(epoch.Get(), msgbyte) //序列化后加入CacheMsg
		return
	}

	//p := fmt.Sprintf("[%v] Handling rbc send message from node %v", m.Instance, m.Source)
	//logging.PrintLog(verbose, logging.NormalLog, p)
	instancestatus.Insert(m.Instance, int(STATUS_SEND))

	msg := m
	msg.Source = id
	msg.Mtype = message.RBC_ECHO
	msg.Value = nil
	tmpmsg := message.WRBCMessage{ //tmpmsg存消息载荷，wrbc不发消息载荷本身
		Value: m.Value,
	}
	tmp, _ := tmpmsg.Serialize()

	msg.Hash = cryptolib.GenInstanceHash(utils.IntToBytes(m.Instance), tmp)
	if !received.IsTrue(m.Instance) {
		receivedReq.Insert(m.Instance, msg.Hash)
		receivedList.Insert(m.Instance, m.Value)
		received.AddItem(m.Instance)
	}

	msgbyte, err := msg.Serialize()
	if err != nil {
		log.Fatalf("failed to serialize echo message")
	}
	sender.MACBroadcast(msgbyte, message.RBC)

	v, exist := cachestatus.Get(m.Instance)
	if exist && v >= int(STATUS_ECHO) {
		SendReady(m)
	}
	if exist && v == int(STATUS_READY) {
		Deliver(m)
	}
}

func HandleEcho(m message.WRBCMessage) {
	result, exist := rstatus.Get(m.Instance)
	if exist && result {
		return
	}

	//p := fmt.Sprintf("[%v] Handling echo message from node %v", m.Instance, m.Source)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	hash := utils.BytesToString(m.Hash)
	quorum.Add(m.Source, hash, nil, quorum.PP)

	if quorum.CheckQuorum(hash, quorum.PP) {
		if !received.IsTrue(m.Instance) {
			receivedReq.Insert(m.Instance, m.Hash)
			received.AddItem(m.Instance)
		}
		SendReady(m)
	}
}

func SendReady(m message.WRBCMessage) {
	elock.Lock()
	stat, _ := instancestatus.Get(m.Instance)

	if stat == int(STATUS_SEND) {
		instancestatus.Insert(m.Instance, int(STATUS_ECHO))
		elock.Unlock()
		//p := fmt.Sprintf("Sending ready for instance id %v", m.Instance)
		//logging.PrintLog(verbose, logging.NormalLog, p)

		msg := m
		msg.Source = id
		msg.Mtype = message.RBC_READY
		msgbyte, err := msg.Serialize()
		if err != nil {
			log.Fatalf("failed to serialize ready message")
		}
		sender.MACBroadcast(msgbyte, message.RBC)
	} else {
		v, exist := cachestatus.Get(m.Instance)
		elock.Unlock()
		if exist && v == int(STATUS_READY) {
			instancestatus.Insert(m.Instance, int(STATUS_ECHO))
			Deliver(m)
		} else {
			cachestatus.Insert(m.Instance, int(STATUS_ECHO))
		}
	}
}

func HandleReady(m message.WRBCMessage) {
	result, exist := rstatus.Get(m.Instance)
	if exist && result {
		return
	}

	//p := fmt.Sprintf("[%v] Handling ready message from node %v", m.Instance, m.Source)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	hash := utils.BytesToString(m.Hash)
	quorum.Add(m.Source, hash, nil, quorum.CM)

	if quorum.CheckEqualSmallQuorum(hash) {
		if !received.IsTrue(m.Instance) {
			receivedReq.Insert(m.Instance, m.Hash)
			received.AddItem(m.Instance)
		}
		SendReady(m)
	}

	if quorum.CheckQuorum(hash, quorum.CM) {
		Deliver(m)
	}
}

func Deliver(m message.WRBCMessage) {
	rlock.Lock()
	stat, _ := instancestatus.Get(m.Instance)

	if stat == int(STATUS_ECHO) {
		if !received.IsTrue(m.Instance) {
			receivedReq.Insert(m.Instance, m.Hash)
			received.AddItem(m.Instance)
		}

		//verify whether receivedList is correct 提交前要检查与receivedList是否一致
		if receivedList.Get(m.Instance) != nil {
			tmpmsg := message.WRBCMessage{
				Value: receivedList.Get(m.Instance),
			}
			tmp, _ := tmpmsg.Serialize()
			localhash := cryptolib.GenInstanceHash(utils.IntToBytes(m.Instance), tmp)
			if bytes.Compare(localhash, m.Hash) != 0 {
				log.Printf("inconsistent hashes. Reset receivedList")
				receivedList.Insert(m.Instance, nil)
			}
		}

		instancestatus.Insert(m.Instance, int(STATUS_READY))
		rlock.Unlock()

		//p := fmt.Sprintf("[%v] WRBC Deliver the request epoch %v, curEpoch %v", m.Instance, m.Epoch, epoch.Get())
		//logging.PrintLog(verbose, logging.NormalLog, p)

		rstatus.Insert(m.Instance, true)

	} else {
		rlock.Unlock()
		cachestatus.Insert(m.Instance, int(STATUS_READY))
	}
}

func CacheMsg(e int, msg []byte) {
	cachedMsg.Insert(e, msg)
}
