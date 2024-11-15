package consensus

import (
	"acs/src/broadcast/ecrbc"
	"acs/src/broadcast/wrbc"
	"acs/src/logging"
	"acs/src/mvba/constmvba"
	"acs/src/quorum"
	"acs/src/utils"
	"fmt"
	"sync"
	"time"
)

var fulllist utils.Vec
var checker utils.IntValue
var started bool
var finalvec utils.Vec
var llock sync.RWMutex

var fint1 int64
var fint2 int64

func ProtoConstMonitorRBCStatus(e int) {

	for {
		if epoch.Get() > e { // 为当前纪元监测
			return
		}

		for i := 0; i < n; i++ { // 遍历每个节点
			llock.RLock()
			curCount := astatus.GetCount()          // 获取astatus的计数
			instanceid := GetInstanceID(members[i]) // 获取成员i在当前epoch的实例ID
			status := ecrbc.QueryStatus(instanceid) // 查询成员i的广播实例是否完成
			llock.RUnlock()

			if !astatus.GetStatus(instanceid) && status { // 如果成员i的广播实例尚未被astatus确认，但其RBC已经完成
				llock.Lock()
				astatus.Insert(instanceid, true) // 将成员i的广播实例标记进astatus
				fulllist.Set(instanceid%n, 1)    // 数组中该位置模n的数据改成1
				if astatus.GetCount() > curCount {
					// log.Printf("curcount %v, astatus.GetCount() %v, fulllist %v", curCount, astatus.GetCount(), fulllist.Get())
					constmvba.UpdateList(fulllist.Get()) // 更新constmvba里的参数
				}
				llock.Unlock()
			}

			if astatus.GetCount() >= quorum.QuorumSize() { // 超过n-f个被astatus确认
				// llock.Lock()

				// llock.Unlock()

				elock.Lock()
				if !started { // 启动 MVBA 阶段
					// log.Printf("fulllist %v", fulllist.Get())
					StartConstMVBAPhase(e)
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

func StartConstMVBAPhase(e int) {
	constmvba.StartMVBA(GetInstanceID(iid), e, fulllist.Get())
	go MonitorConstMVBAStatus(e)
}

func MonitorConstMVBAStatus(e int) { // 收尾阶段
	for {
		if epoch.Get() > e {
			return
		}

		status := constmvba.QueryStatus(e)
		// log.Printf("###status %v", status)
		if status {
			val := constmvba.QueryValue(e) // 获取e的raba decide的数组
			WaitUntilReceived(val)         // 等待rbc完成这个e val=[1 0 1 1]
			outputSize.Set(len(val))
			curStatus.Set(READY)

			fint2 = utils.MakeTimestamp()

			p := fmt.Sprintf("finish ProtoConst,time:%v", fint2-fint1)
			logging.PrintLog(verbose, logging.NormalLog, p)

			ExitEpoch()
			constmvba.InitParametersForEpoch(epoch.Get(), GetInstanceIDsOfEpoch())
			return
		}

		time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
	}
}

func WaitUntilReceived(vec []int) {
	allids := GetInstanceIDsOfEpoch() // allids=[8 9 10 11]
	for {
		for i := 0; i < len(vec); i++ {
			if vec[i] == 1 {
				ins := allids[i]
				if ecrbc.QueryReq(ins) != nil || ecrbc.QueryStatus(ins) {
					// if ecrbc.QueryStatus(ins) == true{
					finalvec.Set(i, 1) // Could be ecrbc decode error
				}
			}
		}
		if CheckVecStatus(vec, finalvec.Get()) {
			return
		}
	}
}

func CheckVecStatus(vec1 []int, vec2 []int) bool {
	for i := 0; i < len(vec1); i++ {
		if vec1[i] == 1 && vec2[i] == 0 {
			return false
		}
	}
	return true
}

func StartProtoConst(data []byte) {
	p := fmt.Sprintf("starting ProtoConst epoch %v, instanceid %v, input %v", epoch.Get(), GetInstanceID(iid), Msgdata(data))
	logging.PrintLog(verbose, logging.NormalLog, p)

	fint1 = utils.MakeTimestamp()

	t1 = utils.MakeTimestamp()
	// InitProtoConst()
	rbc_t1 := utils.MakeTimestamp()
	constmvba.GetRBCtT1(rbc_t1)

	ecrbc.StartECRBC(GetInstanceID(iid), data) // 广播
	go ProtoConstMonitorRBCStatus(epoch.Get()) // 监控 RBC 协议的状态

}

func InitProtoConst() {
	InitStatus(n) // e+1

	// constmvba.InitParametersForEpoch(epoch.Get(), GetInstanceIDsOfEpoch())

	ecrbc.InitECRBC(id, n, verbose)
	wrbc.InitRBC(id, n, verbose, sleepTimerValue)

	wrbc.SetEpoch(epoch.Get())
	ecrbc.SetEpoch(epoch.Get())

	elock.Lock()
	started = false
	elock.Unlock()

	fulllist.Init(n)
	finalvec.Init(n)
	checker.Init()
}
