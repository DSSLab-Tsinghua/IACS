package constmvba

import (
	aba "acs/src/aba/pisa"
	"acs/src/broadcast/wrbc"
	"acs/src/quorum"
	"acs/src/utils"
	"log"
	"sync"
	"time"
)

var epoch utils.IntValue // epoch number
var id int64
var iid int
var n int
var verbose bool
var members []int
var sleepTimerValue int
var mapMembers map[int]int
var allids []int
var shuffledids []int

// ///newly added for constmvba
var curList []int
var insList []int
var llock sync.RWMutex
var receivedstatus utils.BoolValue
var committedhash []byte
var aba_t1 int64
var aba_t2 int64
var rbc_t1 int64
var rbc_t2 int64
var rbc_t3 int64

func QueryStatus(instanceid int) bool {
	llock.RLock()
	v, exist := finalstatus.Get(instanceid)
	llock.RUnlock()
	return exist && v == int(STATUS_TERMINATE)
}

func QueryValue(instanceid int) []int {
	llock.RLock()
	defer llock.RUnlock()
	return decidedvalue.Get(instanceid)
}

func UpdateList(input []int) {
	llock.Lock()
	defer llock.Unlock()
	tmp := make([]int, len(input))
	curList = tmp
	insList = tmp
	for i := 0; i < len(input); i++ {
		curList[i] = input[i] // 0 1 2 3
		insList[i] = input[i]
	}

	for i := 0; i < len(input); i++ {
		if input[i] == 1 {
			insList[i] = allids[i] // 8 9 10 11
		}
	}
	wrbc.UpdateList(insList) // 8 9 10 11
}

func StartMVBA(instanceid int, e int, input []int) {
	log.Printf("starting const mvba epoch %v, instanceid %v, input %v", e, instanceid, input)

	rbc_t3 = utils.MakeTimestamp()

	wrbc.SetEpoch(e)
	go wrbc.IterateCachedMsg(e)      // 处理cache消息
	wrbc.StartRBC(instanceid, input) // 广播消息
	go MonitorWRBCStatus(e)          // 监控RBC状态
}

func MonitorWRBCStatus(e int) {
	for {
		if epoch.Get() > e {
			return
		}

		for i := 0; i < n; i++ {
			instanceid := allids[i]
			status := wrbc.QueryStatus(instanceid)
			if !astatus.GetStatus(instanceid) && status { // 如果当前成员尚未被astatus确认，但其RBC已经提交
				llock.Lock()
				astatus.Insert(instanceid, true)
				RabaReprop(instanceid, e) // RABA
				llock.Unlock()
			}
			if astatus.GetCount() >= quorum.QuorumSize() { // 超过n-f个WRBC已经完成
				ProceedToIteration(e)
				return
			}
		}
		time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
	}
}

func InitParameters() {
	// epoch.Init()
	finalstatus.Init()
	// wrbc.InitRBC(id, n, verbose, sleepTimerValue)
	decidedvalue.Init()
	astatus.Init()
}

func InitParametersForEpoch(e int, instanceids []int) {
	// if epoch.Get() < e {
	// 	epoch.Set(e)
	// } else {
	// 	return
	// }
	epoch.Set(e)
	allids = instanceids
	// var tmp []int
	// curList = tmp

	// wrbc.InitRBC(id, n, verbose, sleepTimerValue)
	astatus.Init()
}

func InitMVBA(thisid int64, numNodes int, ver bool, mem []int, st int) {
	id = thisid
	iid, _ = utils.Int64ToInt(id)
	n = numNodes

	verbose = ver
	quorum.StartQuorum(n)
	members = mem
	sleepTimerValue = st

	round.Init()
	curK.Init()

	// initialize round numbers to 0 for all instances
	mapMembers = make(map[int]int)
	for i := 0; i < len(members); i++ {
		mapMembers[members[i]] = i
	}

	InitParameters()

	baseinstance = 1000 // hard-code to 1000 to avoid conflicts

	InitPer()

	aba.InitABA(id, n, verbose, members, sleepTimerValue)
	aba.InitCoinType(true)
}
