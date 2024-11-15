package cryptolib

import (
	"acs/src/logging"
	"acs/src/utils"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"strings"
	"sync"
)

var randSign = "22220316zafes20180lk7zafes20180619zafepikas"

var randKey = "lk0f7279c18d439459435s714797c9680335a320"

var PriKey *ecdsa.PrivateKey
var PubKey *ecdsa.PublicKey

var MapOfKeys Int64KeyMap

func StartECDSA(id int64, n int) {
	SetHomeDir()
	LoadPrivKeyFromFile(id)
	MapOfKeys.Init()
	for i := 0; i < n; i++ {
		nid := utils.IntToInt64(i)
		LoadPubKeyFromFile(nid)
	}
}

func GenSig(id int64, msg []byte) []byte {

	r, s, err := ecdsa.Sign(strings.NewReader(randSign), PriKey, msg)
	if err != nil {
		log.Println(err)
	}
	signature := r.Bytes()
	signature = append(signature, s.Bytes()...)
	return signature
}

func VerifySig(id int64, msg []byte, sig []byte) bool {
	result := false
	if len(sig) < 56 {
		return false
	}
	r1 := sig[:28]
	s2 := sig[28:]
	r := new(big.Int).SetBytes(r1)
	s := new(big.Int).SetBytes(s2)
	pubKey, exist := MapOfKeys.Get(id)
	if !exist {
		pubKey = LoadPubKeyFromFile(id)
	}
	if pubKey == nil {
		log.Printf("pubkey error %v", id)
		return false //todo: double check
	}
	result = ecdsa.Verify(pubKey, msg, r, s)
	return result
}

func LoadPubKeyFromFile(id int64) *ecdsa.PublicKey {
	path := GenPath(id)
	var pubfileName = fmt.Sprintf("pub.key")

	pubkB, err := ioutil.ReadFile(path + pubfileName)
	if err != nil {
		p := fmt.Sprintf("[ecdsa.go]:LoadPubKeyFromFiles open pub file error! errorinfo:%v\n", err)
		logging.PrintLog(false, logging.ErrorLog, p)
		return nil
	}

	i, err := x509.ParsePKIXPublicKey(pubkB)
	if err != nil {
		log.Printf("error when parsing public key of %v", id)
		return nil
	}

	pubKey, ok := i.(*ecdsa.PublicKey)
	if !ok {
		log.Printf("public conversion error")
		return nil
	}
	MapOfKeys.Insert(id, pubKey)
	return pubKey
}

func LoadPrivKeyFromFile(id int64) {
	path := GenPath(id)
	var privfileName = fmt.Sprintf("priv.key")

	privK, err := ioutil.ReadFile(path + privfileName)
	if err != nil {
		p := fmt.Sprintf("[ecdsa.go]:LoadPrivKeyFromFiles open priv file error! errorinfo:%v\n", err)
		logging.PrintLog(false, logging.ErrorLog, p)
		return
	}
	var errr error
	PriKey, errr = x509.ParseECPrivateKey(privK)
	if errr != nil {
		log.Printf("[ecdsa.go] conversion error %v", err)
		return
	}
}

func GenerateKey(id int64) {
	// generate key
	path := GenPath(id)
	var privfileName = fmt.Sprintf("priv.key")
	var pubfileName = fmt.Sprintf("pub.key")

	if !IsExist(path) {
		err := CreateDir(path)
		if err != nil {
			p := fmt.Sprintf("[ecdsa.go]:GenerateKey create file error! errorinfo:%v\n", err)
			logging.PrintLog(false, logging.ErrorLog, p)
		}
		fmt.Println("creating " + path)
	}

	lenth := len(randKey)
	if lenth < 224/8 {
		log.Printf("[ecdsa error] length of randkey is too short")
		return
	}

	var curve elliptic.Curve
	if lenth > 521/8+8 {
		curve = elliptic.P521()
	} else if lenth > 384/8+8 {
		curve = elliptic.P384()
	} else if lenth > 256/8+8 {
		curve = elliptic.P256()
	} else if lenth > 224/8+8 {
		curve = elliptic.P224()
	}
	// generate the key
	priKey, err := ecdsa.GenerateKey(curve, strings.NewReader(randKey))
	if err != nil {
		return
	}
	// *****************store the key*******************
	// serialize
	priBytes, err := x509.MarshalECPrivateKey(priKey)
	if err != nil {
		return
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&priKey.PublicKey)
	if err != nil {
		return
	}

	err = ioutil.WriteFile(path+privfileName, priBytes, 0644)
	if err != nil {
		p := fmt.Sprintf("[ecdsa.go]:GenerateKey write privKey error! errorinfo:%v\n", err)
		logging.PrintLog(false, logging.ErrorLog, p)
		return
	}
	err = ioutil.WriteFile(path+pubfileName, pubBytes, 0644)
	if err != nil {
		p := fmt.Sprintf("[ecdsa.go]:GenerateKey write pubKey error! errorinfo:%v\n", err)
		logging.PrintLog(false, logging.ErrorLog, p)
		return
	}

}

type Int64KeyMap struct {
	m map[int64]*ecdsa.PublicKey
	sync.RWMutex
}

func (s *Int64KeyMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int64]*ecdsa.PublicKey)
}

func (s *Int64KeyMap) Get(key int64) (*ecdsa.PublicKey, bool) {
	s.RLock()
	defer s.RUnlock()
	_, exist := s.m[key]
	if exist {
		return s.m[key], true
	}
	return nil, false
}

func (s *Int64KeyMap) GetAll() map[int64]*ecdsa.PublicKey {
	return s.m
}

func (s *Int64KeyMap) Insert(key int64, value *ecdsa.PublicKey) {
	s.Lock()
	defer s.Unlock()
	s.m[key] = value
}

func (s *Int64KeyMap) Delete(key int64) {
	s.Lock()
	defer s.Unlock()
	_, exist := s.m[key]
	if exist {
		delete(s.m, key)
	}
}
