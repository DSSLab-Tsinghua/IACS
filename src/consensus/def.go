package consensus

import (
	pisa "acs/src/aba/pisa"
	"acs/src/broadcast/ecrbc"
	"acs/src/broadcast/rbc"
	"acs/src/communication/sender"
	"acs/src/config"
	"acs/src/cryptolib"
	"acs/src/indexacs"
	"acs/src/mvba/constmvba"
	"acs/src/utils"
	"log"
	"sync"
)

type ConsensusType int

const (
	PACE       ConsensusType = 6
	ProtoConst ConsensusType = 17
	ACS        ConsensusType = 100
)

type RbcType int

const (
	RBC   RbcType = 0
	ECRBC RbcType = 1
)

func StartProcessing(data []byte) {
	switch consensus {
	case PACE:
		StartPACEBFT(data, true)
	case ProtoConst: // FIN
		StartProtoConst(data)
	case ACS:
		StartACS(data)
	}

}

func GetInstanceID(input int) int {
	return input + n*epoch.Get() // baseinstance*epoch.Get()
}

func GetIndexFromInstanceID(input int, e int) int {
	return input - n*e
}

func GetInstanceIDsOfEpoch() []int {
	var output []int
	for i := 0; i < len(members); i++ {
		output = append(output, GetInstanceID(members[i]))
	}
	return output
}

func StartHandler(rid string) {
	id, errs = utils.StringToInt64(rid)

	if errs != nil {
		log.Printf("[Error] Replica id %v is not valid. Double check the configuration file", id)
		return
	}
	iid, _ = utils.StringToInt(rid) // 读取整数id

	config.LoadConfig()
	cryptolib.StartCrypto(id, config.CryptoOption()) // 0
	consensus = ConsensusType(config.Consensus())    // 17
	rbcType = RbcType(config.RBCType())              // 1
	n = config.FetchNumReplicas()                    // 50000
	curStatus.Init()                                 // READY
	epoch.Init()                                     // 0
	queue.Init()                                     // []
	verbose = config.FetchVerbose()                  // true
	sleepTimerValue = config.FetchSleepTimer()       // 60

	nodes := config.FetchNodes() // 0123
	for i := 0; i < len(nodes); i++ {
		nid, _ := utils.StringToInt(nodes[i])
		members = append(members, nid) // 将转换后的节点ID添加到成员列表members中
	}

	log.Printf("sleeptimer value %v", sleepTimerValue)
	switch consensus {
	case PACE:
		log.Printf("running PACE")
		InitPACEBFT(true)

		if rbcType == RBC {
			log.Printf("running RBC")
			rbc.InitRBC(id, n, verbose)
		} else if rbcType == ECRBC {
			log.Printf("running ECRBC")
			ecrbc.InitECRBC(id, n, verbose)
		}
		pisa.InitABA(id, n, verbose, members, sleepTimerValue)
	case ProtoConst: // FIN:RBC+MVBA
		log.Printf("running Const")
		InitProtoConst()
		// ecrbc.InitECRBC(id, n, verbose)
		constmvba.InitParametersForEpoch(epoch.Get(), GetInstanceIDsOfEpoch())
		constmvba.InitMVBA(id, n, verbose, members, sleepTimerValue)
	case ACS:
		log.Printf("running ACS")
		InitACS() // e+1
		indexacs.InitParametersForEpoch(epoch.Get(), GetInstanceIDsOfEpoch())
		indexacs.Initindexacs(id, n, verbose, members, sleepTimerValue)
	default:
		log.Fatalf("Consensus type not supported")
	}

	sender.StartSender(rid) // 启动发送者服务
	go RequestMonitor()     // 在新的goroutine中启动请求监控服务
}

type QueueHead struct {
	Head string
	sync.RWMutex
}

func (c *QueueHead) Set(head string) {
	c.Lock()
	defer c.Unlock()
	c.Head = head
}

func (c *QueueHead) Get() string {
	c.RLock()
	defer c.RUnlock()
	return c.Head
}

type CurStatus struct {
	enum Status
	sync.RWMutex
}

type Status int

const (
	READY      Status = 0
	PROCESSING Status = 1
)

func (c *CurStatus) Set(status Status) {
	c.Lock()
	defer c.Unlock()
	c.enum = status
}

func (c *CurStatus) Init() {
	c.Lock()
	defer c.Unlock()
	c.enum = READY
}

func (c *CurStatus) Get() Status {
	c.RLock()
	defer c.RUnlock()
	return c.enum
}
