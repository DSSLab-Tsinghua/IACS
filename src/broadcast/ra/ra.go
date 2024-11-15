package ra

import (
	"acs/src/communication/sender"
	"acs/src/message"
	"acs/src/quorum"
	"acs/src/utils"
	"log"
	"sync"
)

type RAStatus int

const ( //代表不同阶段的消息
	STATUS_IDLE  RAStatus = 0
	STATUS_ECHO  RAStatus = 1
	STATUS_READY RAStatus = 2
)

var rstatus utils.IntBoolMap       // broadcast status,only has value when RBC Deliver 广播实例是否完成
var instancestatus utils.IntIntMap // status for each instance, used in RBC 广播实例当前的阶段
var cachestatus utils.IntIntMap    // status for each instance
var receivedReq utils.IntByteMap   // req is serialized RawOPS or replica msg 广播实例的payload
var received utils.IntSet          // 已经收到的广播实例集合
var elock sync.Mutex               // 互斥锁
var rlock sync.Mutex               // 互斥锁
var receivedSig utils.IntByteMap   // 广播实例的sig
var ulock sync.RWMutex
var insList utils.IntBoolMap

// check whether the instance has been deliver in RBC
func QueryStatus(instanceid int) bool { //查询广播实例是否已经完成
	v, exist := rstatus.Get(instanceid)
	return v && exist
}

func QueryStatusCount() int { //查询已完成的广播实例的个数
	return rstatus.GetCount()
}

func QueryReq(instanceid int) []byte { //查询广播实例的req消息
	v, exist := receivedReq.Get(instanceid)
	if !exist {
		return nil
	}
	return v
}

func QueryECHOStatus(instanceid int) bool { //查询广播实例是否已经进入ECHO阶段且有签名
	stat, _ := instancestatus.Get(instanceid)
	v, _ := receivedSig.Get(instanceid)
	return stat >= int(STATUS_ECHO) && v != nil
}

func QuerySig(instanceid int) []byte { //查询广播实例的签名
	v, _ := receivedSig.Get(instanceid)
	return v
}

func HandleEcho(m message.ReplicaMessage) { //处理Echo消息
	result, exist := rstatus.Get(m.Instance)
	if exist && result { //广播实例已经完成，退出
		return
	}

	//p := fmt.Sprintf("[%v] RA Handling echo message from node %v", m.Instance, m.Source)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	hash := utils.BytesToString(m.Hash)
	quorum.Add(m.Source, hash, nil, quorum.PP) //将消息的哈希部分加入quorum.PP

	if quorum.CheckQuorum(hash, quorum.PP) { //收到过超过n-f个与此消息相同的哈希
		if !received.IsTrue(m.Instance) && !wrbc { //之前没收到过此广播实例且不是wrbc
			receivedReq.Insert(m.Instance, m.Payload) //payload加入receivedReq
			received.AddItem(m.Instance)              //标记为收到过
		}
		SendReady(m) //发送Ready
	}
}

func SendReady(m message.ReplicaMessage) { //发送Ready
	elock.Lock()
	stat, _ := instancestatus.Get(m.Instance)
	if stat == int(STATUS_ECHO) {
		return
	}
	instancestatus.Insert(m.Instance, int(STATUS_ECHO)) //进入ECHO阶段
	elock.Unlock()

	//p := fmt.Sprintf("RA Sending ready for instance id %v", m.Instance)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	msg := m
	msg.Source = id
	msg.Mtype = message.RA_READY
	msgbyte, err := msg.Serialize()
	if err != nil {
		log.Fatalf("failed to serialize ready message")
	}
	sender.MACBroadcast(msgbyte, message.RA) //构建Ready消息序列化广播

}

func HandleReady(m message.ReplicaMessage) { //处理Ready消息
	result, exist := rstatus.Get(m.Instance)
	if exist && result { //广播实例已经完成，退出
		return
	}

	//p := fmt.Sprintf("[%v] RA Handling ready message from node %v", m.Instance, m.Source)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	hash := utils.BytesToString(m.Hash)
	quorum.Add(m.Source, hash, nil, quorum.CM) //将消息的哈希部分加入quorum.CM

	if quorum.CheckEqualSmallQuorum(hash) { //f+1个相同哈希
		if !received.IsTrue(m.Instance) { //之前没收到过此广播实例
			receivedReq.Insert(m.Instance, m.Payload)
			received.AddItem(m.Instance)
		}
		SendReady(m) //发Ready
	}

	if quorum.CheckQuorum(hash, quorum.CM) { //超n-f个相同哈希，提交
		Deliver(m)
	}
}

func Deliver(m message.ReplicaMessage) {

	if !received.IsTrue(m.Instance) && !wrbc { //之前没收到过此广播实例且不是wrbc
		receivedReq.Insert(m.Instance, m.Payload)
		received.AddItem(m.Instance)
	}

	//p := fmt.Sprintf("[%v] RA Deliver the request epoch %v, curEpoch %v", m.Instance, m.Epoch, epoch.Get())
	//logging.PrintLog(verbose, logging.NormalLog, p)

	rstatus.Insert(m.Instance, true) //完成

}
