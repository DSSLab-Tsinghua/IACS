package indexgather

import (
	"acs/src/communication"
	"acs/src/communication/sender"
	"acs/src/config"
	"acs/src/logging"
	"acs/src/message"
	"acs/src/quorum"
	"acs/src/utils"
	"fmt"
	"log"
	"time"
)

func HandleInform(m message.ReplicaMessage) {
	S := utils.BytesToInts(m.Payload)
	v := m.Instance
	//p := fmt.Sprintf("[%v] Handling inform message from node %v", m.Instance, m.Source)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	for {
		if Ifinclude(v, S) {
			msg := m
			msg.Source = id
			msg.Mtype = message.IG_ACK
			msg.Payload = nil

			// msg := message.ReplicaMessage{
			//	Mtype:    message.IG_ACK,
			//	Instance: v,
			//	Source:   id,
			//	TS:       utils.MakeTimestamp(),
			//	Payload:  nil,
			//	Epoch:    e,
			// }

			msgbyte, err := msg.Serialize()
			if err != nil {
				log.Fatalf("failed to serialize ack message")
			}

			nodes := sender.FetchNodesFromConfig()
			nid := nodes[m.Source]
			dest, _ := utils.StringToInt64(nid)
			request, err := message.SerializeWithMAC(id, dest, msgbyte)
			if err != nil {
				logging.PrintLog(true, logging.ErrorLog, "[Sender Error] Not able to generate MAC")
				continue
			}
			if communication.IsNotLive(nid) {
				p := fmt.Sprintf("[Communication Sender] Replica %v is not live, don't send message to it", nid)
				logging.PrintLog(verbose, logging.NormalLog, p)
				continue
			}
			go sender.ByteSend(request, config.FetchAddress(nid), message.IG_ALL)
			break
		}
		time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
	}

}

func HandleAck(m message.ReplicaMessage) {
	//p := fmt.Sprintf("[%v] Handling ack message from node %v", m.Instance, m.Source)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	quorum.Add(m.Source, utils.IntToString(m.Instance), nil, quorum.PP) // 将消息的哈希部分加入quorum.PP
	if quorum.CheckQuorum(utils.IntToString(m.Instance), quorum.PP) {   // 收到过超过n-f个与此消息相同的哈希
		msg := message.ReplicaMessage{
			Mtype:    message.IG_PREPARE,
			Instance: m.Instance,
			Source:   id,
			TS:       utils.MakeTimestamp(),
			Payload:  utils.IntsToBytes(valid.Get(m.Instance)),
			Epoch:    m.Epoch,
		}

		msgbyte, err := msg.Serialize()
		if err != nil {
			log.Fatalf("failed to serialize IG message")
		}
		sender.MACBroadcast(msgbyte, message.IG)
	}
}

func HandlePrepare(m message.ReplicaMessage) {
	T := utils.BytesToInts(m.Payload)
	v := m.Instance
	thisid, _ := utils.Int64ToInt(m.Source)

	//p := fmt.Sprintf("[%v] Handling prepare message from node %v", m.Instance, m.Source)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	for {
		if Ifinclude(v, T) {
			tmp := setvalue.Get(m.Instance)
			tmp = append(tmp, thisid)
			setvalue.Insert(m.Instance, tmp)
			quorum.Add(m.Source, utils.IntToString(m.Instance), nil, quorum.CM)
			if quorum.CheckQuorum(utils.IntToString(m.Instance), quorum.CM) {
				decidedvalue.Insert(m.Instance, setvalue.Get(m.Instance))
				finalstatus.Insert(m.Instance, true)
			}
		}
		time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
	}
}

func Ifinclude(insid int, vec []int) bool {
	if Ifinclude_vec(vec, valid.Get(insid)) {
		return true
	}
	return false
}

func Ifinclude_vec(vec []int, valid []int) bool {
	if len(vec) == 0 || len(valid) == 0 || len(vec) != len(valid) {
		return false
	}
	for i := 0; i < len(vec); i++ {
		if vec[i] == 1 && valid[i] == 0 {
			return false
		}
	}
	return true
}
