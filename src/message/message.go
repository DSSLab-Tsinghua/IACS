//各种消息的定义、序列化、解序列化函数

package message

import (
	"acs/src/cryptolib"
	//"encoding/json"
	pb "acs/src/proto/proto/communication"

	"github.com/vmihailenco/msgpack"
)

type ReplicaMessage struct {
	Mtype    TypeOfMessage
	Instance int
	Source   int64
	Hash     []byte
	TS       int64
	Payload  []byte //message payload
	Sig      []byte
	Value    int
	Maj      int
	Round    int
	Epoch    int
}

type MessageWithSignature struct {
	Msg []byte
	Sig []byte
}

type RawOPS struct {
	OPS []pb.RawMessage
}

type FPCCMessage struct {
	FPCC [][]byte
}

type CBCMessage struct {
	Value         map[int][]byte
	RawData       [][]byte
	MerkleBranch  [][][]byte
	MerkleIndexes [][]int64
}

type WRBCMessage struct {
	Mtype    TypeOfMessage
	Instance int
	Source   int64
	Hash     []byte
	Value    []int
	Epoch    int
}

type FragMessage struct {
	RawData       []byte
	MerkleBranch  [][]byte
	MerkleIndexes []int64
}

type Signatures struct {
	Hash []byte
	Sigs [][]byte
	IDs  []int64
}

func (r *FPCCMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeFPCCMessage(input []byte) FPCCMessage {
	var fpcc = new(FPCCMessage)
	msgpack.Unmarshal(input, &fpcc)
	return *fpcc
}

func (r *FragMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeFragMessage(input []byte) FragMessage {
	var fragMessage = new(FragMessage)
	msgpack.Unmarshal(input, &fragMessage)
	return *fragMessage
}

func (r *Signatures) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeSignatures(input []byte) ([]byte, [][]byte, []int64) {
	var sigs = new(Signatures)
	msgpack.Unmarshal(input, &sigs)
	return sigs.Hash, sigs.Sigs, sigs.IDs
}

func (r *CBCMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeCBCMessage(input []byte) CBCMessage {
	var cbcMessage = new(CBCMessage)
	msgpack.Unmarshal(input, &cbcMessage)
	return *cbcMessage
}

func (r *WRBCMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeWRBCMessage(input []byte) WRBCMessage {
	var wrbcmessage = new(WRBCMessage)
	msgpack.Unmarshal(input, &wrbcmessage)
	return *wrbcmessage
}

func (r *RawOPS) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

/*
Deserialize []byte to MessageWithSignature
*/
func DeserializeRawOPS(input []byte) RawOPS {
	var rawOPS = new(RawOPS)
	msgpack.Unmarshal(input, &rawOPS)
	return *rawOPS
}

/*
Serialize MessageWithSignature
*/
func (r *MessageWithSignature) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

/*
Deserialize []byte to MessageWithSignature
*/
func DeserializeMessageWithSignature(input []byte) MessageWithSignature {
	var messageWithSignature = new(MessageWithSignature)
	msgpack.Unmarshal(input, &messageWithSignature)
	return *messageWithSignature
}

func (r *ReplicaMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeReplicaMessage(input []byte) ReplicaMessage {
	var replicaMessage = new(ReplicaMessage)
	msgpack.Unmarshal(input, &replicaMessage)
	return *replicaMessage
}

func SerializeWithSignature(id int64, msg []byte) ([]byte, error) {
	request := MessageWithSignature{
		Msg: msg,
		Sig: cryptolib.GenSig(id, msg),
	}

	requestSer, err := request.Serialize()
	if err != nil {
		return []byte(""), err
	}
	return requestSer, err
}

func SerializeWithMAC(id int64, dest int64, msg []byte) ([]byte, error) {
	request := MessageWithSignature{
		Msg: msg,
		Sig: cryptolib.GenMAC(id, msg),
	}

	requestSer, err := request.Serialize()
	if err != nil {
		return []byte(""), err
	}
	return requestSer, err
}
