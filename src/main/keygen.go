/*
generating threshold prf keys
*/

package main

import (
	prf "acs/src/cryptolib/threshprf"
	"acs/src/utils"
	"flag"
	"log"
	"os"
)

const (
	helpText_Keygen = `
Generating keys for threshold prf
keygen [n] [k]
n: number of replicas, k: threshold
`
)

func main() {

	helpPtr := flag.Bool("help", false, helpText_Keygen)
	flag.Parse()

	if *helpPtr || len(os.Args) < 2 {
		log.Printf(helpText_Keygen)
		return
	}

	n, _ := utils.StringToInt64(os.Args[1])
	k, _ := utils.StringToInt64(os.Args[2])

	log.Printf("[Keygen] Generating keys for n=%v, k=%v.", n, k)
	prf.SetHomeDir()
	prf.Init_key_dealer(n, k)

}
