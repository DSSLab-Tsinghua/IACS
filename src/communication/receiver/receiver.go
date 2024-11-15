/*
Receiver functions.
It implements all the gRPC services defined in communication.proto file.
*/

package receiver

import (
	coin "acs/src/aba/coin"
	pisa "acs/src/aba/pisa"
	"acs/src/broadcast/ecrbc"
	"acs/src/broadcast/ra"
	"acs/src/broadcast/rbc"
	"acs/src/broadcast/rbc2"
	"acs/src/broadcast/rbc3"
	"acs/src/broadcast/rbc4"
	"acs/src/broadcast/wrbc"
	"acs/src/communication"
	"acs/src/config"
	"acs/src/consensus"
	"acs/src/cryptolib"
	"acs/src/indexgather"
	logging "acs/src/logging"
	"acs/src/mvba/constmvba"
	pb "acs/src/proto/proto/communication"
	"acs/src/utils"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"google.golang.org/grpc"
)

var id string
var wg sync.WaitGroup
var sleepTimerValue int
var con int

type server struct {
	pb.UnimplementedSendServer
}

type reserver struct {
	pb.UnimplementedSendServer
}

/*
Handle replica messages (consensus normal operations)
*/
func (s *server) SendMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	return &pb.Empty{}, nil
}

func (s *server) SendRequest(ctx context.Context, in *pb.Request) (*pb.RawMessage, error) {
	return HandleRequest(in)
}

func (s *reserver) SendRequest(ctx context.Context, in *pb.Request) (*pb.RawMessage, error) {
	return HandleRequest(in)
}

func HandleRequest(in *pb.Request) (*pb.RawMessage, error) {
	wtype := in.GetType()
	switch wtype {
	case pb.MessageType_WRITE_BATCH:
		consensus.HandleBatchRequest(in.GetRequest())
		reply := []byte("batch rep")

		return &pb.RawMessage{Msg: reply}, nil
	default:
		h := cryptolib.GenHash(in.GetRequest())
		go consensus.HandleRequest(in.GetRequest(), utils.BytesToString(h))

		reply := []byte("rep")

		return &pb.RawMessage{Msg: reply}, nil
	}

}

func (s *server) RBCSendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	switch consensus.ConsensusType(con) {
	case consensus.ProtoConst:
		go wrbc.HandleRBCMsg(in.GetMsg())
	default:
		go rbc.HandleRBCMsg(in.GetMsg())
	}
	return &pb.Empty{}, nil
}

func (s *server) ECRBCSendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	go ecrbc.HandleECRBCMsg(in.GetMsg())
	return &pb.Empty{}, nil
}

func (s *server) MVBASendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	switch consensus.ConsensusType(con) {
	case consensus.ProtoConst:
		go constmvba.HandleDistributeMsg(in.GetMsg())
	}

	return &pb.Empty{}, nil
}

func (s *server) ABASendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	switch consensus.ConsensusType(con) {
	case consensus.PACE, consensus.ProtoConst:
		go pisa.HandleABAMsg(in.GetMsg())
	default:
		log.Fatalf("consensus type %v not supported in ABASendByteMsg function", con)
	}

	return &pb.Empty{}, nil
}

func (s *server) PRFSendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	go coin.HandleCoinMsg(in.GetMsg())
	return &pb.Empty{}, nil
}

func (s *server) RASendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	go ra.HandleRAMsg(in.GetMsg())
	return &pb.Empty{}, nil
}

func (s *server) IGSendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	go indexgather.HandleIGMsg(in.GetMsg())
	return &pb.Empty{}, nil
}

func (s *server) RBC2SendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	switch consensus.ConsensusType(con) {
	case consensus.ProtoConst:
		go wrbc.HandleRBCMsg(in.GetMsg())
	default:
		go rbc2.HandleRBCMsg(in.GetMsg())
	}
	return &pb.Empty{}, nil
}
func (s *server) RBC3SendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	switch consensus.ConsensusType(con) {
	case consensus.ProtoConst:
		go wrbc.HandleRBCMsg(in.GetMsg())
	default:
		go rbc3.HandleRBCMsg(in.GetMsg())
	}
	return &pb.Empty{}, nil
}

func (s *server) RBC4SendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	switch consensus.ConsensusType(con) {
	case consensus.ProtoConst:
		go wrbc.HandleRBCMsg(in.GetMsg())
	default:
		go rbc4.HandleRBCMsg(in.GetMsg())
	}
	return &pb.Empty{}, nil
}

/*
Handle join requests for both static membership (initialization) and dynamic membership.
Each replica gets a conformation for a membership request.
*/
func (s *server) Join(ctx context.Context, in *pb.RawMessage) (*pb.RawMessage, error) {

	reply := []byte("hi") // handler.HandleJoinRequest(in.GetMsg())
	result := true

	return &pb.RawMessage{Msg: reply, Result: result}, nil
}

/*
Register rpc socket via port number and ip address
*/
// 用于注册一个 gRPC 服务并开始监听客户端连接，通过端口和配置参数灵活控制网络服务的行为
func register(port string, splitPort bool) {
	lis, err := net.Listen("tcp", port) // 监听 TCP 连接，返回监听器对象 lis

	if err != nil {
		p := fmt.Sprintf("[Communication Receiver Error] failed to listen %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		os.Exit(1)
	}
	if config.FetchVerbose() { // true
		p := fmt.Sprintf("[Communication Receiver] listening to port %v", port)
		logging.PrintLog(config.FetchVerbose(), logging.NormalLog, p)
	}

	log.Printf("ready to listen to port %v", port)
	go serveGRPC(lis, splitPort) // splitPort:false

}

/*
Have serve grpc as a function (could be used together with goroutine)
*/
// 主要用于启动 gRPC 服务器，通过监听指定的端口来处理 RPC 请求
func serveGRPC(lis net.Listener, splitPort bool) {
	defer wg.Done() // 运行完后减少一个goroutine

	if splitPort { // false不使用分割端口模式

		s1 := grpc.NewServer(grpc.MaxRecvMsgSize(52428800), grpc.MaxSendMsgSize(52428800))

		pb.RegisterSendServer(s1, &reserver{})
		log.Printf("listening to split port")
		if err := s1.Serve(lis); err != nil {
			p := fmt.Sprintf("[Communication Receiver Error] failed to serve: %v", err)
			logging.PrintLog(true, logging.ErrorLog, p)
			os.Exit(1)
		}

		return
	}

	s := grpc.NewServer(grpc.MaxRecvMsgSize(52428800), grpc.MaxSendMsgSize(52428800))
	// 创建一个新的 gRPC 服务器实例，能够接收和发送的最大消息大小，均为50MB

	pb.RegisterSendServer(s, &server{}) // 将服务注册到 gRPC 服务器

	if err := s.Serve(lis); err != nil {
		p := fmt.Sprintf("[Communication Receiver Error] failed to serve: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		os.Exit(1)
	}

}

/*
Start receiver parameters initialization
*/
// 启动一个可配置的接收器服务，同时可以启用共识机制和多端口监听，具备并发处理能力
func StartReceiver(rid string, cons bool) {
	id = rid
	logging.SetID(rid)

	config.LoadConfig()                        // 加载配置文件
	logging.SetLogOpt(config.FetchLogOpt())    // 获取日志选项0
	con = config.Consensus()                   // 获取共识机制设置17FIN
	sleepTimerValue = config.FetchSleepTimer() // 获取睡眠计时器的值60s

	if cons {
		consensus.StartHandler(rid)
		//log.Printf("StartHandler finish")
	}

	if config.SplitPorts() { // false
		// wg.Add(1)
		go register(communication.GetPortNumber(config.FetchPort(rid)), true)
	}
	wg.Add(1)                              // 增加一个goroutine
	register(config.FetchPort(rid), false) // 注册并启动一个 TCP 服务，使用gRPC
	wg.Wait()                              // 等待所有的goroutine都完成再退出
}
