package ecrbc

import (
	"acs/src/communication/sender"
	"acs/src/message"
	"acs/src/quorum"
	"acs/src/utils"
	"log"
	"sync"
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
var receivedReq utils.IntByteMap   //req is serialized RawOPS or replica msg
var receivedFrag utils.IntBytesMap
var decodedInstance utils.IntBytesMap //decode the erasure for instance upon receive f+1 frags
var decodeStatus utils.Set            //set true if decode
var entireInstance utils.IntByteMap   //set the decoded instance payload
var elock sync.Mutex
var rlock sync.Mutex
var qlock sync.Mutex
var decodeLock sync.Mutex
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
	v, exist := entireInstance.Get(instanceid)
	if !exist {
		return nil
	}
	return v
}

func QueryInstanceFrag(instanceid int) []byte {
	v, exist := receivedReq.Get(instanceid)
	if !exist {
		return nil
	}
	return v
}

func HandleSend(m message.ReplicaMessage) {
	result, exist := rstatus.Get(m.Instance)
	if exist && result {
		return
	}

	//p := fmt.Sprintf("[%v] Handling ECRBC send message from node %v", m.Instance, m.Source)
	//logging.PrintLog(verbose, logging.NormalLog, p)
	rlock.Lock()
	instancestatus.Insert(m.Instance, int(STATUS_SEND))
	rlock.Unlock()

	msg := m
	msg.Source = id
	msg.Mtype = message.RBC_ECHO
	//msg.Hash = cryptolib.GenInstanceHash(utils.IntToBytes(m.Instance),m.Payload)
	receivedReq.Insert(m.Instance, m.Payload)

	msgbyte, err := msg.Serialize()
	if err != nil {
		log.Fatalf("failed to serialize echo message")
	}
	var data [][]byte
	data = append(data, msgbyte)

	sender.MACBroadcastWithErasureCode(data, message.ECRBC, false)

	v, exist := cachestatus.Get(m.Instance)
	if exist && v >= int(STATUS_ECHO) {
		SendReady(m)
	}
	if exist && v == int(STATUS_READY) {
		Deliver(m)
	}
}

func HandleEcho(m message.ReplicaMessage) {
	result, exist := rstatus.Get(m.Instance)
	if exist && result {
		return
	}

	//p := fmt.Sprintf("[%v] Handling ECRBC echo message from node %v", m.Instance, m.Source)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	receivedFrag.InsertValueAndInt(m.Instance, m.Payload, m.Source)

	hash := utils.IntToString(m.Instance)
	quorum.Add(m.Source, hash, nil, quorum.PP)

	if quorum.CheckSmallQuorum(hash, quorum.PP) {
		decodeLock.Lock()
		ErasureDecoding(m.Instance, quorum.SQuorumSize(), quorum.NSize())
		decodeLock.Unlock()
	}

	if quorum.CheckQuorum(hash, quorum.PP) {
		SendReady(m)
	}
}

func SendReady(m message.ReplicaMessage) {
	elock.Lock()

	stat, _ := instancestatus.Get(m.Instance)

	if stat == int(STATUS_SEND) {
		instancestatus.Insert(m.Instance, int(STATUS_ECHO))
		elock.Unlock()
		//p := fmt.Sprintf("Sending ready for instance id %v", m.Instance)
		//logging.PrintLog(verbose, logging.NormalLog, p)

		var msgs [][]byte
		var frag []byte

		msg := m
		msg.Source = id
		msg.Mtype = message.RBC_READY
		selfIndex, _ := utils.Int64ToInt(id)

		frag = QueryInstanceFrag(m.Instance)

		if frag == nil {
			data, exi := decodedInstance.Get(m.Instance)
			if !exi {
				log.Fatalf("%v has noe been decoded", m.Instance)
			}
			frag = data[selfIndex]
		}

		msg.Payload = frag
		msgbyte, err := msg.Serialize()
		if err != nil {
			log.Fatalf("failed to serialize RBC message")
		}
		msgs = append(msgs, msgbyte)

		sender.MACBroadcastWithErasureCode(msgs, message.ECRBC, false)

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

func HandleReady(m message.ReplicaMessage) {
	result, exist := rstatus.Get(m.Instance)
	if exist && result {
		return
	}
	qlock.Lock()
	defer qlock.Unlock()
	hash := utils.IntToString(m.Instance)
	quorum.Add(m.Source, hash, nil, quorum.CM)

	//p := fmt.Sprintf("[%v] Handling ercbc ready message from node %v, curnum %v", m.Instance, m.Source, quorum.CheckCurNum(hash, quorum.CM))
	//logging.PrintLog(verbose, logging.NormalLog, p)

	if quorum.CheckSmallQuorum(hash, quorum.CM) {
		decodeLock.Lock()
		ErasureDecoding(m.Instance, quorum.SQuorumSize(), quorum.NSize())
		decodeLock.Unlock()
	}

	if quorum.CheckEqualSmallQuorum(hash) {
		SendReady(m)
	}

	if quorum.CheckQuorum(hash, quorum.CM) {
		Deliver(m)
	}
}

func Deliver(m message.ReplicaMessage) {
	rlock.Lock()
	stat, _ := instancestatus.Get(m.Instance)

	//p := fmt.Sprintf("[%v] ECRBC Pre-Deliver the request epoch %v, curEpoch %v", m.Instance, m.Epoch, epoch.Get())
	//logging.PrintLog(verbose, logging.NormalLog, p)

	if stat == int(STATUS_ECHO) {
		instancestatus.Insert(m.Instance, int(STATUS_READY))

		//p := fmt.Sprintf("[%v] ECRBC Deliver the request epoch %v, curEpoch %v", m.Instance, m.Epoch, epoch.Get())
		//logging.PrintLog(verbose, logging.NormalLog, p)

		rstatus.Insert(m.Instance, true)
		rlock.Unlock()

	} else {
		rlock.Unlock()
		//p := fmt.Sprintf("[%v] ECRBC cache the request epoch %v, curEpoch %v", m.Instance, m.Epoch, epoch.Get())
		//logging.PrintLog(verbose, logging.NormalLog, p)
		cachestatus.Insert(m.Instance, int(STATUS_READY))
	}
}

func CacheMsg(e int, msg []byte) {
	cachedMsg.Insert(e, msg)
}
