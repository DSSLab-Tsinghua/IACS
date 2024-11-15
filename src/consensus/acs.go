package consensus

import (
	"acs/src/broadcast/ra"
	"acs/src/broadcast/rbc"
	"acs/src/broadcast/rbc2"
	"acs/src/broadcast/rbc3"
	"acs/src/broadcast/rbc4"
	"acs/src/indexacs"
	"acs/src/logging"
	"acs/src/message"
	"acs/src/quorum"
	"acs/src/utils"
	"fmt"
	"time"
)

var acst1 int64
var acst2 int64

func ACSMonitorRBCStatus(e int) {
	for {
		if epoch.Get() > e { // 为当前纪元监测
			return
		}

		for i := 0; i < n; i++ { // 遍历每个节点
			llock.RLock()
			curCount := astatus.GetCount()          // 获取astatus的计数
			instanceid := GetInstanceID(members[i]) // 获取成员i在当前epoch的实例ID
			status := rbc.QueryStatus(instanceid)   // 查询成员i的广播实例是否完成
			llock.RUnlock()

			if !astatus.GetStatus(instanceid) && status { // 如果成员i的广播实例尚未被astatus确认，但其RBC已经完成
				llock.Lock()
				astatus.Insert(instanceid, true)   // 将成员i的广播实例标记进astatus
				fulllist.Set(instanceid%n, 1)      // 数组中该位置模n的数据改成1
				if astatus.GetCount() > curCount { // 如果astatus的计数增加，更新共识模块的成员列表状态
					// log.Printf("curcount %v, astatus.GetCount() %v, fulllist %v", curCount, astatus.GetCount(), fulllist.Get())
					indexacs.UpdateList(fulllist.Get())
				}
				llock.Unlock()
			}

			if astatus.GetCount() >= quorum.QuorumSize() { // 超过n-f个被astatus确认
				elock.Lock()
				if !started { // 启动 indexacs 阶段
					StartIndexACSPhase(e)
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

func StartIndexACSPhase(e int) {
	indexacs.Startindexacs(GetInstanceID(iid), e, fulllist.Get()) // valid
	go MonitorindexacsStatus(e)
}

func MonitorindexacsStatus(e int) { // 收尾阶段，要indexACS的finalstatus和decidedvalue
	for {
		if epoch.Get() > e {
			return
		}

		status := indexacs.QueryStatus() // finalstatus结束

		if status {
			val := indexacs.QueryValue() // indexacs的输出X decidedvalue 数组，索引数组
			WaitUntilReceived_acs(val)
			outputSize.Set(len(val))
			curStatus.Set(READY)

			list := output.SetList()
			var msg []string
			for _, by := range list {
				msg = append(msg, Msgdata(by))
			}

			acst2 = utils.MakeTimestamp()

			p := fmt.Sprintf("finish acs,output:%v,total time:%v", msg, acst2-acst1)
			logging.PrintLog(verbose, logging.NormalLog, p)

			return
		}

		time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
	}
}

func WaitUntilReceived_acs(vec []int) {
	allids := GetInstanceIDsOfEpoch()
	for {
		for i := 0; i < len(vec); i++ {
			if vec[i] == 1 {
				ins := allids[i]
				if rbc.QueryReq(ins) != nil || rbc.QueryStatus(ins) {
					finalvec.Set(i, 1)
					output.AddItem(rbc.QueryReq(ins)) // output为ACS结果
				}
			}
		}
		if CheckVecStatus_acs(vec, finalvec.Get()) { // acs输出的全部完成rbc
			return
		}
	}
}

func CheckVecStatus_acs(vec1 []int, vec2 []int) bool {
	for i := 0; i < len(vec1); i++ {
		if vec1[i] == 1 && vec2[i] == 0 {
			return false
		}
	}
	return true
}

func StartACS(data []byte) {
	p := fmt.Sprintf("starting acs epoch %v, instanceid %v, input %v", epoch.Get(), GetInstanceID(iid), Msgdata(data))
	logging.PrintLog(verbose, logging.NormalLog, p)

	acst1 = utils.MakeTimestamp()

	rbc.StartRBC(GetInstanceID(iid), data)
	go ACSMonitorRBCStatus(epoch.Get())
}

func InitACS() {
	InitStatus(n) // e+1

	rbc.InitRBC(id, n, verbose)
	rbc2.InitRBC(id, n, verbose)
	rbc3.InitRBC(id, n, verbose)
	rbc4.InitRBC(id, n, verbose)
	ra.InitRA(id, n, verbose)

	rbc.SetEpoch(epoch.Get())
	rbc2.SetEpoch(epoch.Get())
	rbc3.SetEpoch(epoch.Get())
	rbc4.SetEpoch(epoch.Get())
	ra.SetEpoch(epoch.Get())

	elock.Lock()
	started = false
	elock.Unlock()

	fulllist.Init(n)
	finalvec.Init(n)
}

func Msgdata(data []byte) string {
	rops := message.DeserializeRawOPS(data)
	request := rops.OPS[0].Msg
	requestSer := request
	dataSer := message.DeserializeMessageWithSignature(requestSer).Msg
	op := message.DeserializeClientRequest(dataSer).OP
	msg := string(op)
	return msg
}
