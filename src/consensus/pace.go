package consensus

import (
	aba "acs/src/aba/pisa"
	"acs/src/broadcast/ecrbc"
	"acs/src/broadcast/rbc"
	"acs/src/logging"
	"acs/src/quorum"
	"acs/src/utils"
	"fmt"
	"log"
	"sync"
	"time"
)

var aba_t1 int64
var aba_t2 int64
var rbc_t1 int64
var rbc_t2 int64
var aba_started bool
var lllock sync.Mutex

func PACEMonitorRBCStatus(e int) {
	for {
		if epoch.Get() > e {
			return
		}

		for i := 0; i < n; i++ {
			instanceid := GetInstanceID(members[i])
			status := rbc.QueryStatus(instanceid)

			if !astatus.GetStatus(instanceid) && status {
				astatus.Insert(instanceid, true)
				go PACEStartABA(instanceid, 1)
			}

			switch consensus {
			case PACE:
				if astatus.GetCount() >= quorum.QuorumSize() {
					go PACEStartOtherABAs()
				}
			}

		}
		time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
	}
}

func PACEMonitorECRBCStatus(e int) {
	for {
		if epoch.Get() > e {
			return
		}

		for i := 0; i < n; i++ {
			instanceid := GetInstanceID(members[i])
			status := ecrbc.QueryStatus(instanceid)

			if !astatus.GetStatus(instanceid) && status {
				astatus.Insert(instanceid, true)
				lllock.Lock()
				if !aba_started {
					aba_t1 = utils.MakeTimestamp()
					aba_started = true
				}
				lllock.Unlock()
				go PACEStartABA(instanceid, 1)
			}

			switch consensus {
			case PACE:
				if astatus.GetCount() >= quorum.QuorumSize() {
					rbc_t2 = utils.MakeTimestamp()
					go PACEStartOtherABAs()
				}
			}

		}
		time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
	}
}

func PACEMonitorABAStatus(e int) {
	for {
		if epoch.Get() > e {
			return
		}

		for i := 0; i < n; i++ {
			instanceid := GetInstanceID(members[i])
			status := aba.QueryStatus(instanceid)

			if !fstatus.GetStatus(instanceid) && status {
				fstatus.Insert(instanceid, true)
				//log.Printf("instance %v completed", instanceid)
				go PACEUpdateOutput(instanceid)
			}

			if fstatus.GetCount() == n {
				aba_t2 = utils.MakeTimestamp()
				if aba_t1 != 0 && aba_t2-aba_t1 >= int64(sleepTimerValue) && rbc_t2-rbc_t1 >= int64(sleepTimerValue) {
					log.Printf("RBC:RABA %v %v", rbc_t2-rbc_t1, aba_t2-aba_t1)
					p := fmt.Sprintf("RBC:RABA %v %v", rbc_t2-rbc_t1, aba_t2-aba_t1)
					logging.PrintLog(true, logging.BreakdownLog, p)
				}
				return
			}

		}
		time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
	}
}

func PACEUpdateOutput(instanceid int) {
	value := aba.QueryValue(instanceid)
	//log.Printf("+++++++++++++++++finish [%v]",instanceid)
	if value == 0 {
		outputCount.Increment()
	} else {
		outputSize.Increment()
		for {
			var v []byte
			if rbcType == RBC {
				v = rbc.QueryReq(instanceid)
			} else if rbcType == ECRBC {
				v = ecrbc.QueryReq(instanceid)
			}
			//if v != nil{
			output.AddItem(v)
			outputCount.Increment()
			break
			//}else{
			//time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
			//}
		}
	}

	if outputCount.Get() == n {
		curStatus.Set(READY)
		ExitEpoch()
		/*switch consensus{
		case ACE:
			InitPACEBFT(false)
		case PACE:
			InitPACEBFT(true)
		}*/
		return
	}
}

func PACEStartABA(instanceid int, input int) {
	if bstatus.GetStatus(instanceid) {
		return
	}
	bstatus.Insert(instanceid, true)
	switch consensus {
	case PACE:
		aba.StartABAFromRoundZero(instanceid, input)
	default:
		log.Fatalf("This script only supports ACE and PACE")
	}

}

func PACEStartOtherABAs() {
	//log.Printf("Start other ABAs")
	if otherlock.Get() == 1 {
		return
	}
	for i := 0; i < n; i++ {
		instanceid := GetInstanceID(members[i])
		if !astatus.GetStatus(instanceid) {
			go PACEStartABA(instanceid, 0)
		}
	}
	otherlock.Set(1)
}

func StartPACEBFT(data []byte, ct bool) {
	t1 = utils.MakeTimestamp()
	InitPACEBFT(ct)
	if rbcType == RBC {
		rbc.StartRBC(GetInstanceID(iid), data)
		go PACEMonitorRBCStatus(epoch.Get())
	} else if rbcType == ECRBC {
		rbc_t1 = utils.MakeTimestamp()
		aba_started = false
		ecrbc.StartECRBC(GetInstanceID(iid), data)
		go PACEMonitorECRBCStatus(epoch.Get())

	}

	go PACEMonitorABAStatus(epoch.Get())
}

func InitPACEBFT(ct bool) {
	//rbc.InitRBC(id,n,verbose)
	//aba.InitABA(id,n,verbose,members,sleepTimerValue)
	aba.InitCoinType(ct)
	InitStatus(n)

}
