package client

import (
	sender "acs/src/communication/clientsender"
	"acs/src/config"

	logging "acs/src/logging"
	"acs/src/message"
	pb "acs/src/proto/proto/communication"
	"acs/src/utils"
	// "encoding/json"
	"fmt"
	"log"

	"github.com/vmihailenco/msgpack"
)

var cid int64
var err error
var clientTimer int

func GetCID() int64 {
	return cid
}

func SignedRequest(cid int64, dataSer []byte) ([]byte, bool) { // 签名消息并序列化
	request := message.MessageWithSignature{
		Msg: dataSer,
		Sig: []byte(""), // cryptolib.GenSig(cid, dataSer),
	}

	requestSer, err := request.Serialize()
	if err != nil {
		p := fmt.Sprintf("[Client error] fail to serialize the request with signiture: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return []byte(""), false
	}

	return requestSer, true
}

func SendWriteRequest(op []byte) { // hi

	dataSer, result1 := CreateRequest(cid, op) // 创建消息并序列化
	if !result1 {
		return
	}

	requestSer, result2 := SignedRequest(cid, dataSer) // 签名消息并序列化
	if !result2 {
		return
	}
	log.Println("len of request: ", len(requestSer))
	sender.BroadcastRequest(pb.MessageType_WRITE, requestSer)
}

func SendBatchRequest(op []byte, batchSize int) {
	var requestArr [][]byte
	for i := 0; i < batchSize; i++ {

		dataSer, result1 := CreateRequest(cid, op)
		if !result1 {
			return
		}

		requestSer, result2 := SignedRequest(cid, dataSer)
		if !result2 {
			return
		}
		// log.Println("len of request in batch: ",len(requestSer))
		requestArr = append(requestArr, requestSer)
	}
	byteRequsets, err := SerializeRequests(requestArr)
	if err != nil {
		log.Fatal("[Client error] fail to serialize the message.")
	}

	sender.BroadcastRequest(pb.MessageType_WRITE_BATCH, byteRequsets)
}

func CreateRequest(cid int64, op []byte) ([]byte, bool) { // 创建消息并序列化
	data := message.ClientRequest{
		Type: pb.MessageType_WRITE,
		ID:   cid,
		OP:   op,
		TS:   utils.MakeTimestamp(),
	}

	dataSer, err := data.Serialize()
	if err != nil {
		p := fmt.Sprintf("[Client error] fail to serialize the write request: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return []byte(""), false
	}

	return dataSer, true
}

/*
Serialize data into a json object in bytes
Output

	[]byte: serialized request
	error: err is nil if request is serialized
*/
func SerializeRequests(r [][]byte) ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		p := fmt.Sprintf("[Client error] fail to serialize the message %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return []byte(""), err
	}
	return jsons, nil
}

func StartClient(rid string, loadkey bool) {
	logging.SetID(rid)
	config.LoadConfig()
	logging.SetLogOpt(config.FetchLogOpt())

	log.Printf("Client %s started.", rid)
	cid, err = utils.StringToInt64(rid)
	sender.StartClientSender(rid, loadkey)
	clientTimer = config.FetchBroadcastTimer()
}
