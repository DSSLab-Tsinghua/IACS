package indexacs

import (
	"acs/src/broadcast/rbc2"
	"acs/src/indexvaba"
	"acs/src/quorum"
	"acs/src/utils"
	"sync"
	"time"
)

var llock sync.RWMutex
var elock sync.Mutex

var valid utils.Vec
var epoch utils.IntValue // epoch number
var allids []int
var id int64
var iid int
var n int
var verbose bool
var members []int
var sleepTimerValue int

var firstpre bool
var started bool

var astatus utils.IntBoolMap
var fulllist utils.Vec
var finalstatus bool
var decidedvalue []int

var indexacst1 int64
var indexacst2 int64

// 要完善finalstatus和decidedvalue的值，让acs读取
func QueryStatus() bool {
	return finalstatus
}

func QueryValue() []int {
	return decidedvalue
}

func UpdateList(input []int) {
	for i, value := range input {
		valid.Set(i, value)
	}
}

func Startindexacs(instanceid int, e int, input []int) {
	indexacst1 = utils.MakeTimestamp()

	//p := fmt.Sprintf("starting index acs epoch %v, instanceid %v, input %v", e, instanceid, input)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	rbc2.StartRBC(instanceid, utils.IntsToBytes(input)) // 广播消息
	go MonitorRBC2Status(e)                             // 监控RBC状态
}

func MonitorRBC2Status(e int) {
	for {
		if epoch.Get() > e {
			return
		}

		for i := 0; i < n; i++ {
			llock.RLock()
			curCount := astatus.GetCount() // 获取astatus的计数
			instanceid := allids[i]
			status := rbc2.QueryStatus(instanceid)
			llock.RUnlock()

			if !astatus.GetStatus(instanceid) && status {
				I := utils.BytesToInts(rbc2.QueryReq(instanceid))
				if Ifinclude(I) && Number1count(I) >= quorum.QuorumSize() {
					llock.Lock()
					if !firstpre {
						pre := instanceid % n
						indexvaba.Setpre(pre)
						firstpre = true
					}
					astatus.Insert(instanceid, true)
					fulllist.Set(instanceid%n, 1)
					if astatus.GetCount() > curCount { // 如果astatus的计数增加，更新共识模块的成员列表状态
						// log.Printf("curcount %v, astatus.GetCount() %v, fulllist %v", curCount, astatus.GetCount(), fulllist.Get())
						indexvaba.UpdateList(fulllist.Get())
					}
					llock.Unlock()
				}
			}

			if astatus.GetCount() >= quorum.QuorumSize() { // 超过n-f个被astatus确认
				elock.Lock()
				if !started { // 启动 indexacs 阶段
					StartIndexVABAPhase(e)
					started = true // 只启动一次
				}
				elock.Unlock()
			}

			elock.Lock()
			if astatus.GetCount() == n { // 全部被astatus确认，退出
				elock.Unlock()
				return
			}
			elock.Unlock()
		}
		time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
	}
}

func StartIndexVABAPhase(e int) {
	indexvaba.Startindexvaba(GetInstanceID(iid), e, fulllist.Get()) // valid
	go MonitorindexvabaStatus(e)
}

func MonitorindexvabaStatus(e int) { // vaba的收尾阶段，要indexVABA的finalstatus和decidedvalue
	for {
		if epoch.Get() > e {
			return
		}

		status := indexvaba.QueryStatus() // finalstatus结束
		// log.Printf("###status %v", status)
		if status {
			index := indexvaba.QueryValue() // indexvaba的输出X decidedvalue 一个索引
			if rbc2.QueryStatus(allids[index]) {
				decidedvalue = utils.BytesToInts(rbc2.QueryReq(allids[index]))
				finalstatus = true

				indexacst2 = utils.MakeTimestamp()

				//p := fmt.Sprintf("finish indexACS,output:%v,time:%v", decidedvalue, indexacst2-indexacst1)
				//logging.PrintLog(verbose, logging.NormalLog, p)
				return
			}
		}
		time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
	}
}

func Ifinclude(vec []int) bool {
	for i := 0; i < len(vec); i++ {
		if vec[i] == 1 && valid.GetIdx(i) == 0 {
			return false
		}
	}
	return true
}

func Number1count(vec []int) int {
	count := 0
	for i := 0; i < len(vec); i++ {
		if vec[i] == 1 {
			count++
		}
	}
	return count
}

// func GetInstanceIDsOfEpoch() []int {
//	var output []int
//	for i := 0; i < len(members); i++ {
//		output = append(output, GetInstanceID(members[i]))
//	}
//	return output
// }

func GetInstanceID(input int) int {
	return input + n*epoch.Get() // baseinstance*epoch.Get()
}

func InitParametersForEpoch(e int, instanceids []int) {
	epoch.Set(e)
	allids = instanceids
	indexvaba.InitParametersForEpoch(epoch.Get())
}

func Initindexacs(thisid int64, numNodes int, ver bool, mem []int, st int) {
	id = thisid
	iid, _ = utils.Int64ToInt(id)
	n = numNodes

	verbose = ver
	quorum.StartQuorum(n)
	members = mem
	sleepTimerValue = st

	fulllist.Init(n)
	astatus.Init()
	valid.Init(n)

	indexvaba.Initindexvaba(id, n, verbose, members, sleepTimerValue)
}
