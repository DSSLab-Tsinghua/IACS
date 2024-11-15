/*
Main function for servers. Start a replica.
*/

package main

import (
	"acs/src/communication/receiver"
	"flag"
	"log"
	"os"
)

const (
	helpText_Server = `
Main function for servers. Start a replica. 
server [ReplicaID]
`
)

func main() {
	helpPtr := flag.Bool("help", false, helpText_Server) // 如果命令行中包含-help参数，helpPtr会被设置为true
	flag.Parse()                                         // 解析命令行参数

	if *helpPtr || len(os.Args) < 2 { // 检查helpPtr指向的值是否为true，或者命令行参数的数量是否少于2个
		log.Printf(helpText_Server) // 打印帮助信息
		return
	}

	id := "0"
	if len(os.Args) > 1 { // 检查命令行参数的数量是否2个及以上
		id = os.Args[1] // 将id变量的值设置为第二个参数
	}

	log.Printf("**Starting replica %s", id) // 打印一条日志信息，表明程序正在启动，后面跟着id变量的值
	receiver.StartReceiver(id, true)        // 调用receiver包中的StartReceiver函数，传入id和true作为参数

}
