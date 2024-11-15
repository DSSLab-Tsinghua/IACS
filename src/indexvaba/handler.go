package indexvaba

import (
	"acs/src/broadcast/ra"
	"acs/src/broadcast/rbc3"
	"acs/src/broadcast/rbc4"
	"acs/src/indexgather"
	"acs/src/quorum"
	"acs/src/utils"
	"log"
	"sort"
	"sync"
	"time"
)

var llock sync.RWMutex
var lllock sync.RWMutex
var elock sync.Mutex
var eelock sync.Mutex
var rastarted bool

var valid utils.Vec
var epoch utils.IntValue
var id int64
var iid int
var n int
var verbose bool
var members []int
var sleepTimerValue int

var view utils.IntIntMap
var pre int

var justify utils.IntIntMap

var M utils.IntIntMapArr

var igvalid utils.Vec

var finalstatus bool
var decidedvalue int

var indexvabat1 int64
var indexvabat2 int64

var finished bool

type prejustify struct {
	pre     int
	justify map[int]int
}

// 要完善finalstatus和decidedvalue的值，让indexacs读取
func QueryStatus() bool {
	return finalstatus
}

func QueryValue() int {
	return decidedvalue
}

func UpdateList(input []int) {
	for i, value := range input {
		valid.Set(i, value)
	}
}

func Setpre(input int) {
	pre = input
}

func GetInstanceIDforview(input int, view int) int {
	return input + n*(epoch.Get()+view)
}

func Startindexvaba(instanceid int, e int, input []int) {
	indexvabat1 = utils.MakeTimestamp()

	//p := fmt.Sprintf("starting index vaba epoch %v, instanceid %v, input %v", e, instanceid, input)
	//logging.PrintLog(verbose, logging.NormalLog, p)
	// log.Printf("starting index vaba epoch %v, instanceid %v, input %v", e, instanceid, input)
	ProceedToIteration(e)
}

func ProceedToIteration(e int) {
	var message utils.Prejustify
	message.SetPrejustify(pre, justify.GetAll())
	// go MonitorMStatus(e)
	StartLoop(e, message)
}

func StartLoop(e int, message utils.Prejustify) {
	v, _ := view.Get(e)
	if v >= n {
		log.Printf("Faltal error, number of iterations is longer than n")
		return
	}

	rbc3.StartRBC(GetInstanceIDforview(iid, v), message.PrejustifyToBytes())
	go MonitorRBC3Status(e, v)
}

func MonitorRBC3Status(e int, v int) {
	var astatus utils.IntBoolMap
	astatus.Init()
	var started bool
	indexgather.Initindexgather(id, n, verbose, members, sleepTimerValue, v) // 所有人参与同一个，只和view有关

	for {
		if epoch.Get() > e {
			return
		}
		for i := 0; i < n; i++ {
			llock.RLock()
			curCount := astatus.GetCount()
			instanceid := GetInstanceIDforview(i, v)
			status := rbc3.QueryStatus(instanceid)
			llock.RUnlock()

			if !astatus.GetStatus(instanceid) && status {

				data := utils.BytesToPrejustify(rbc3.QueryReq(instanceid))
				if Ifinclude_pre(data.Getpre()) && Ifinclude_justify(data.Getjustify(), v-1) {
					llock.Lock()
					if v > 1 {
						if len(data.Getjustify()) < quorum.QuorumSize() || !Isfrequent(data.Getpre(), data.Getjustify()) {
							continue
						}
					}
					astatus.Insert(instanceid, true)
					igvalid.Set(instanceid%n, 1)
					if astatus.GetCount() > curCount { // 如果astatus的计数增加，更新共识模块的成员列表状态
						indexgather.UpdateList(igvalid.Get(), v)
					}
					llock.Unlock()
				}
			}

			if astatus.GetCount() >= quorum.QuorumSize() { // 超过n-f个被astatus确认
				elock.Lock()
				if !started { // 启动 indexacs 阶段
					StartIndexGatherPhase(e, v, igvalid.Get())
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

func StartIndexGatherPhase(e int, v int, valid []int) {
	indexgather.Startindexgather(e, v, valid)
	go MonitorindexgatherStatus(e, v)
}

func MonitorindexgatherStatus(e int, v int) {
	for {
		if epoch.Get() > e {
			return
		}

		status := indexgather.QueryStatus(v) // finalstatus结束
		// log.Printf("###status %v", status)
		if status {
			val := indexgather.QueryValue(v) // indexvaba的输出X decidedvalue 索引数组
			// 求最高等级的索引
			sort.Ints(val)
			for i, j := 0, len(val)-1; i < j; i, j = i+1, j-1 {
				val[i], val[j] = val[j], val[i]
			}

			l := val[0]
			data_l := utils.BytesToPrejustify(rbc3.QueryReq(GetInstanceIDforview(l, v)))
			pre_l := data_l.Getpre()
			vote := pre_l

			//p := fmt.Sprintf("indexVABA max rank:%v,vote:%v", l, vote)
			//logging.PrintLog(verbose, logging.NormalLog, p)

			rbc4.StartRBC(GetInstanceIDforview(iid, v), utils.IntToBytes(vote))
			go MonitorRBC4Status(e, v)
			return
		}
		time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
	}
}

func MonitorRBC4Status(e int, v int) {
	var astatus utils.IntBoolMap
	astatus.Init()
	var started bool

	for {
		if epoch.Get() > e {
			return
		}

		for i := 0; i < n; i++ {
			lllock.RLock()
			instanceid := GetInstanceIDforview(i, v)
			status := rbc4.QueryStatus(instanceid)
			lllock.RUnlock()

			if !astatus.GetStatus(instanceid) && status {
				vote := utils.BytesToInt(rbc4.QueryReq(instanceid))
				if Ifinigvalid(vote) {
					lllock.Lock()
					astatus.Insert(instanceid, true)
					M.InsertValue(v, i, vote)
					// M[v][i] = vote
					lllock.Unlock()
				}
			}

			if astatus.GetCount() >= quorum.QuorumSize() { // 超过n-f个被astatus确认
				elock.Lock()
				if !started {
					newjustify := M.Get(v).GetAll()
					newpre, _ := Getfrequent(newjustify)
					var message utils.Prejustify
					message.SetPrejustify(newpre, newjustify)
					go MonitorMStatus(e, v)
					view.Increment(e)
					StartLoop(e, message)
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

func MonitorMStatus(e int, v int) {
	if epoch.Get() > e {
		return
	}
	for {
		tmp := M.Get(v).GetAll()
		k, count := Getfrequent(tmp)
		if count >= quorum.QuorumSize() {
			eelock.Lock()
			if !rastarted {
				ra.StartRA(e, utils.IntToBytes(k))
				go MonitorRAStatus(e)
				rastarted = true
			}
			eelock.Unlock()
			return
		}
		time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
	}
}

func MonitorRAStatus(e int) {
	if epoch.Get() > e {
		return
	}
	for {
		if ra.QueryStatus(e) {
			indexvabat2 = utils.MakeTimestamp()
			decidedvalue = utils.BytesToInt(ra.QueryReq(e))
			finalstatus = true

			if !finished {
				//p := fmt.Sprintf("finish indexVABA,output:%v,time:%v", decidedvalue, indexvabat2-indexvabat1)
				//logging.PrintLog(verbose, logging.NormalLog, p)
				finished = true
			}

		}
		time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
	}
}

func Ifinclude_pre(pre int) bool {
	return valid.GetIdx(pre) == 1
}

func Ifinclude_justify(justfity map[int]int, view int) bool {
	if view == -1 {
		return true
	}
	for key, _ := range justfity {
		if justfity[key] != M.GetValue(view, key) {
			return false
		}
	}
	return true
}

func Isfrequent(pre int, justify map[int]int) bool {
	count := make(map[int]int)
	maxCount := 0
	for _, value := range justify {
		count[value]++
		if count[value] > maxCount {
			maxCount = count[value]
		}
	}
	return count[pre] == maxCount
}

func Getfrequent(justify map[int]int) (int, int) {
	count := make(map[int]int)
	maxCount := 0
	for _, value := range justify {
		count[value]++
		if count[value] > maxCount {
			maxCount = count[value]
		}
	}
	for num, cnt := range count {
		if cnt == maxCount {
			return num, maxCount
		}
	}
	return -1, -1
}

func Ifinigvalid(vote int) bool {
	return igvalid.GetIdx(vote) == 1
}

func InitParametersForEpoch(e int) {
	epoch.Set(e)
}

func Initindexvaba(thisid int64, numNodes int, ver bool, mem []int, st int) {
	id = thisid
	iid, _ = utils.Int64ToInt(id)
	n = numNodes

	verbose = ver
	quorum.StartQuorum(n)
	members = mem
	sleepTimerValue = st

	igvalid.Init(n)
	justify.Init()
	valid.Init(n)

	M.Init()

	view.Init()
}
