//客户端请求消息

package message

import (
	//"encoding/json"
	pb "acs/src/proto/proto/communication"

	"github.com/vmihailenco/msgpack"
)

type ClientRequest struct {
	Type pb.MessageType
	ID   int64
	OP   []byte // Message payload. Opt for contract.
	TS   int64  // Timestamp
}

func (r *ClientRequest) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeClientRequest(input []byte) ClientRequest {
	var clientRequest = new(ClientRequest)
	msgpack.Unmarshal(input, &clientRequest)
	return *clientRequest
}
