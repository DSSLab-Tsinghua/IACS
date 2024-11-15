package consensus

import (
	"acs/src/logging"
	"acs/src/message"
	"acs/src/quorum"
	"acs/src/utils"
	//"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/vmihailenco/msgpack"
)

var verbose bool //verbose level
var id int64     //id of server
var iid int      //id in type int, start a RBC using it to instanceid
var errs error
var queue Queue         // cached client requests
var sleepTimerValue int // sleeptimer for the while loop that continues to monitor the queue or the request status
var consensus ConsensusType
var rbcType RbcType
var n int
var members []int
var t1 int64

var batchSize int
var requestSize int

func ExitEpoch() {
	t2 := utils.MakeTimestamp()
	if (t2 - t1) == 0 {
		log.Printf("Latancy is zero!")
		return
	}
	if outputSize.Get() == 0 {
		log.Printf("Finish zero instacne!")
		return
	}
	log.Printf("*****epoch %v ends with %v, output size %v, latency %v ms, throughput %d, tps %d", epoch.Get(), output.Len(), outputSize.Get(), t2-t1, int64(outputSize.Get()*batchSize*1000)/(t2-t1), int64(quorum.QuorumSize()*batchSize*1000)/(t2-t1))
	p := fmt.Sprintf("%v %v %v %v %v %v %v", quorum.FSize(), batchSize, int64(outputSize.Get()*batchSize*1000)/(t2-t1), int64(quorum.QuorumSize()*batchSize*1000)/(t2-t1), t2-t1, requestSize, outputSize.Get())
	logging.PrintLog(true, logging.EvaluationLog, p)

}

func CaptureRBCLat() {
	t3 := utils.MakeTimestamp()
	if (t3 - t1) == 0 {
		log.Printf("Latancy is zero!")
		return
	}
	log.Printf("*****RBC phase ends with %v ms", t3-t1)

}

func CaptureLastRBCLat() {
	t3 := utils.MakeTimestamp()
	if (t3 - t1) == 0 {
		log.Printf("Latancy is zero!")
		return
	}
	log.Printf("*****Final RBC phase ends with %v ms", t3-t1)

}

func RequestMonitor() { //用于监控和处理队列中的请求
	for {
		if curStatus.Get() == READY && !queue.IsEmpty() { //当前状态为准备就绪，并且队列不为空
			curStatus.Set(PROCESSING) //当前状态为处理中

			time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond) //60毫秒
			batch := queue.GrabWtihMaxLenAndClear()                       //从队列中获取一批请求，并清空队列
			batchSize = len(batch)
			log.Printf("Handling batch requests with len %v\n", batchSize)
			rops := message.RawOPS{
				OPS: batch, //创建一个 message.RawOPS 类型的对象 rops，并将 batch 作为操作列表赋值给它
			}
			data, err := rops.Serialize() //将 rops 序列化为数据
			if err != nil {
				continue
			}
			StartProcessing(data) //处理序列化后的数据
		} else {
			time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
		}
	}
}

func HandleRequest(request []byte, hash string) {
	//log.Printf("Handling request")
	//rawMessage := message.DeserializeMessageWithSignature(request)
	//m := message.DeserializeClientRequest(rawMessage.Msg)

	/*if !cryptolib.VerifySig(m.ID, rawMessage.Msg, rawMessage.Sig) {
		log.Printf("[Authentication Error] The signature of client request has not been verified.")
		return
	}*/
	//log.Printf("Receive len %v op %v\n",len(request),m.OP)
	batchSize = 1
	requestSize = len(request)
	queue.Append(request)
}

func HandleBatchRequest(requests []byte) {
	requestArr := DeserializeRequests(requests)
	//var hashes []string
	Len := len(requestArr)
	//log.Printf("Handling batch requests with len %v\n",Len)
	//for i:=0;i<Len;i++{
	//	hashes = append(hashes,string(cryptolib.GenHash(requestArr[i])))
	//}
	//for i:=0;i<Len;i++{
	//	HandleRequest(requestArr[i],hashes[i])
	//}
	/*for i:=0;i<Len;i++{
		rawMessage := message.DeserializeMessageWithSignature(requestArr[i])
		m := message.DeserializeClientRequest(rawMessage.Msg)

		if !cryptolib.VerifySig(m.ID, rawMessage.Msg, rawMessage.Sig) {
			log.Printf("[Authentication Error] The signature of client logout request has not been verified.")
			return
		}
	}*/
	batchSize = Len
	requestSize = len(requestArr[0])
	queue.AppendBatch(requestArr)
}

func DeserializeRequests(input []byte) [][]byte {
	var requestArr [][]byte
	msgpack.Unmarshal(input, &requestArr)
	return requestArr
}
