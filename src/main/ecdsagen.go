/*
generating threshold prf keys
*/

package main

import (
	"acs/src/cryptolib"
	"acs/src/utils"
	"flag"
	"log"
	"os"
)

const (
	helpText = `
Generating keys for ecdsa
keygen [initID] [endID]
`
)

func main() {

	helpPtr := flag.Bool("help", false, helpText)
	flag.Parse()

	if *helpPtr || len(os.Args) < 2 {
		log.Printf(helpText)
		return
	}

	initid := "0"
	if len(os.Args) > 1 {
		initid = os.Args[1]
	}

	endid := initid
	if len(os.Args) > 2 {
		endid = os.Args[2]
	}

	bid, _ := utils.StringToInt64(initid)
	eid, _ := utils.StringToInt64(endid)
	cryptolib.SetHomeDir()
	for i := bid; i <= eid; i++ {
		cryptolib.GenerateKey(i)
	}
}
