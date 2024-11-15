package rbc4

import (
	"acs/src/communication/sender"
	"acs/src/cryptolib"
	"acs/src/message"
	"acs/src/quorum"
	"acs/src/utils"
	"log"
	"sync"
)

type RBCStatus int

const ( // 代表不同阶段的消息
	STATUS_IDLE  RBCStatus = 0
	STATUS_SEND  RBCStatus = 1
	STATUS_ECHO  RBCStatus = 2
	STATUS_READY RBCStatus = 3
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
func QueryStatus(instanceid int) bool { // 查询广播实例是否已经完成
	v, exist := rstatus.Get(instanceid)
	return v && exist
}

func QueryStatusCount() int { // 查询已完成的广播实例的个数
	return rstatus.GetCount()
}

func QueryReq(instanceid int) []byte { // 查询广播实例的req消息
	v, exist := receivedReq.Get(instanceid)
	if !exist {
		return nil
	}
	return v
}

func QueryECHOStatus(instanceid int) bool { // 查询广播实例是否已经进入ECHO阶段且有签名
	stat, _ := instancestatus.Get(instanceid)
	v, _ := receivedSig.Get(instanceid)
	return stat >= int(STATUS_ECHO) && v != nil
}

func QuerySig(instanceid int) []byte { // 查询广播实例的签名
	v, _ := receivedSig.Get(instanceid)
	return v
}

func UpdateList(input []int) { // used for external predicate
	ulock.Lock()
	defer ulock.Unlock()
	// log.Printf("-------updated list %v", input)
	for i := 0; i < len(input); i++ {
		insList.Insert(input[i], true)
	}
}
func HandleSend(m message.ReplicaMessage) { // 处理Send消息
	result, exist := rstatus.Get(m.Instance)
	if exist && result { // 广播实例已经完成，退出
		return
	}

	//p := fmt.Sprintf("[%v] rbc4 Handling send message from node %v", m.Instance, m.Source)
	//logging.PrintLog(verbose, logging.NormalLog, p)
	instancestatus.Insert(m.Instance, int(STATUS_SEND)) // 标记为SEND阶段

	msg := m
	msg.Source = id
	msg.Mtype = message.RBC4_ECHO
	msg.Hash = cryptolib.GenInstanceHash(utils.IntToBytes(m.Instance), m.Payload)
	if wrbc { // we do not need the payload WRBC不搭载消息，只传哈希
		m.Payload = nil
	}
	if !received.IsTrue(m.Instance) { // 之前没收到过此广播实例
		receivedReq.Insert(m.Instance, m.Payload) // payload加入receivedReq
		received.AddItem(m.Instance)              // 标记为收到过
		receivedSig.Insert(m.Instance, m.Sig)     // sig加入receivedSig
	}

	msgbyte, err := msg.Serialize() // 将要发送的ECHO消息序列化
	if err != nil {
		log.Fatalf("failed to serialize echo message")
	}
	sender.MACBroadcast(msgbyte, message.RBC4) // 广播给所有人

	v, exist := cachestatus.Get(m.Instance)
	if exist && v >= int(STATUS_ECHO) { // 缓存在ECHO阶段，即调用过SendReady(m)
		SendReady(m) // 循环等待
	}
	if exist && v == int(STATUS_READY) { // 缓存在READY阶段，提交
		Deliver(m)
	}
}

func HandleEcho(m message.ReplicaMessage) { // 处理Echo消息
	result, exist := rstatus.Get(m.Instance)
	if exist && result { // 广播实例已经完成，退出
		return
	}

	//p := fmt.Sprintf("[%v] rbc4 Handling echo message from node %v", m.Instance, m.Source)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	hash := utils.BytesToString(m.Hash)
	quorum.Add(m.Source, hash, nil, quorum.PP) // 将消息的哈希部分加入quorum.PP

	if quorum.CheckQuorum(hash, quorum.PP) { // 收到过超过n-f个与此消息相同的哈希
		if !received.IsTrue(m.Instance) && !wrbc { // 之前没收到过此广播实例且不是wrbc
			receivedReq.Insert(m.Instance, m.Payload) // payload加入receivedReq
			received.AddItem(m.Instance)              // 标记为收到过
		}
		SendReady(m) // 发送Ready
	}
}

func SendReady(m message.ReplicaMessage) { // 发送Ready
	elock.Lock()
	stat, _ := instancestatus.Get(m.Instance)

	if stat == int(STATUS_SEND) { // 处于SEND状态，接收过Send消息
		instancestatus.Insert(m.Instance, int(STATUS_ECHO)) // 进入ECHO阶段
		elock.Unlock()
		//p := fmt.Sprintf(" rbc4 Sending ready for instance id %v", m.Instance)
		//logging.PrintLog(verbose, logging.NormalLog, p)

		msg := m
		msg.Source = id
		msg.Mtype = message.RBC4_READY
		msgbyte, err := msg.Serialize()
		if err != nil {
			log.Fatalf("failed to serialize ready message")
		}
		sender.MACBroadcast(msgbyte, message.RBC4) // 构建Ready消息序列化广播
	} else { // 还没接收过Send消息 or 已经发过Ready消息
		v, exist := cachestatus.Get(m.Instance)
		elock.Unlock()
		if exist && v == int(STATUS_READY) { // 缓存在READY阶段
			instancestatus.Insert(m.Instance, int(STATUS_ECHO)) // 广播进入ECHO阶段，提交
			Deliver(m)
		} else { // 无缓存 or 缓存在ECHO阶段
			cachestatus.Insert(m.Instance, int(STATUS_ECHO)) // 缓存设为ECHO阶段
		}
	}
}

func HandleReady(m message.ReplicaMessage) { // 处理Ready消息
	result, exist := rstatus.Get(m.Instance)
	if exist && result { // 广播实例已经完成，退出
		return
	}

	//p := fmt.Sprintf("[%v] rbc4 Handling ready message from node %v", m.Instance, m.Source)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	hash := utils.BytesToString(m.Hash)
	quorum.Add(m.Source, hash, nil, quorum.CM) // 将消息的哈希部分加入quorum.CM

	if quorum.CheckEqualSmallQuorum(hash) { // f+1个相同哈希
		if !received.IsTrue(m.Instance) { // 之前没收到过此广播实例
			receivedReq.Insert(m.Instance, m.Payload)
			received.AddItem(m.Instance)
		}
		SendReady(m) // 发Ready
	}

	if quorum.CheckQuorum(hash, quorum.CM) { // 超n-f个相同哈希，提交
		Deliver(m)
	}
}

func Deliver(m message.ReplicaMessage) {
	rlock.Lock()
	stat, _ := instancestatus.Get(m.Instance)

	if stat == int(STATUS_ECHO) { // 在ECHO阶段
		if !received.IsTrue(m.Instance) && !wrbc { // 之前没收到过此广播实例且不是wrbc
			receivedReq.Insert(m.Instance, m.Payload)
			received.AddItem(m.Instance)
		}
		instancestatus.Insert(m.Instance, int(STATUS_READY)) // 进入READY阶段
		rlock.Unlock()

		//p := fmt.Sprintf("[%v] rbc4 Deliver the request epoch %v, curEpoch %v", m.Instance, m.Epoch, epoch.Get())
		//logging.PrintLog(verbose, logging.NormalLog, p)

		rstatus.Insert(m.Instance, true) // 完成

	} else { // 广播没进ECHO，即未收到Send消息，只收Ready消息提交的，缓存进入READY阶段
		rlock.Unlock()
		cachestatus.Insert(m.Instance, int(STATUS_READY))
	}
}
