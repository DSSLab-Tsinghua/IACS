// 消息类型定义

package message

type TypeOfMessage int

const (
	RBC_ALL         TypeOfMessage = 1
	RBC_SEND        TypeOfMessage = 2
	RBC_ECHO        TypeOfMessage = 3
	RBC_READY       TypeOfMessage = 4
	ABA_ALL         TypeOfMessage = 5
	ABA_BVAL        TypeOfMessage = 6
	ABA_AUX         TypeOfMessage = 7
	ABA_CONF        TypeOfMessage = 8
	ABA_FINAL       TypeOfMessage = 9
	PRF             TypeOfMessage = 10
	ECRBC_ALL       TypeOfMessage = 11
	CBC_ALL         TypeOfMessage = 12
	CBC_SEND        TypeOfMessage = 13
	CBC_REPLY       TypeOfMessage = 14
	CBC_ECHO        TypeOfMessage = 15
	CBC_EREPLY      TypeOfMessage = 16
	CBC_READY       TypeOfMessage = 17
	MVBA_DISTRIBUTE TypeOfMessage = 18
	EVCBC_ALL       TypeOfMessage = 19
	RETRIEVE        TypeOfMessage = 20
	SIMPLE_SEND     TypeOfMessage = 21
	ECHO_SEND       TypeOfMessage = 22
	ECHO_REPLY      TypeOfMessage = 23
	GC_ALL          TypeOfMessage = 24
	GC_FORWARD      TypeOfMessage = 25
	GC_ECHO         TypeOfMessage = 26
	GC_READY        TypeOfMessage = 27
	GC_CAST         TypeOfMessage = 28
	SIMPLE_PROOF    TypeOfMessage = 29
	PLAINCBC_ALL    TypeOfMessage = 30
	DISPERSE        TypeOfMessage = 31
	MBA_ALL         TypeOfMessage = 32
	MBA_ECHO        TypeOfMessage = 33
	MBA_FORWARD     TypeOfMessage = 34
	MBA_DISTRIBUTE  TypeOfMessage = 35
	RA_ALL          TypeOfMessage = 36
	RA_ECHO         TypeOfMessage = 37
	RA_READY        TypeOfMessage = 38
	IG_ALL          TypeOfMessage = 39
	IG_INFORM       TypeOfMessage = 40
	IG_ACK          TypeOfMessage = 41
	IG_PREPARE      TypeOfMessage = 42
	RBC2_ALL        TypeOfMessage = 43
	RBC2_SEND       TypeOfMessage = 44
	RBC2_ECHO       TypeOfMessage = 45
	RBC2_READY      TypeOfMessage = 46
	RBC3_ALL        TypeOfMessage = 47
	RBC3_SEND       TypeOfMessage = 48
	RBC3_ECHO       TypeOfMessage = 49
	RBC3_READY      TypeOfMessage = 50
	RBC4_ALL        TypeOfMessage = 51
	RBC4_SEND       TypeOfMessage = 52
	RBC4_ECHO       TypeOfMessage = 53
	RBC4_READY      TypeOfMessage = 54
)

type ProtocolType int

const (
	RBC         ProtocolType = 1
	ABA         ProtocolType = 2
	ECRBC       ProtocolType = 3
	CBC         ProtocolType = 4
	EVCBC       ProtocolType = 5
	MVBA        ProtocolType = 6
	SIMPLE      ProtocolType = 7
	ECHO        ProtocolType = 8
	GC          ProtocolType = 9
	SIMPLEPROOF ProtocolType = 10
	PLAINCBC    ProtocolType = 11
	MBA         ProtocolType = 12
	RA          ProtocolType = 13
	IG          ProtocolType = 14
	RBC2        ProtocolType = 15
	RBC3        ProtocolType = 16
	RBC4        ProtocolType = 17
)

type HashMode int

const (
	DEFAULT_HASH HashMode = 0
	MERKLE       HashMode = 1
)
