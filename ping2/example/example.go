package main

import (
	"os"
	"fmt"
	"github.com/nicle-lin/ping/ping2"
)

func main(){
	if len(os.Args) < 2{
		fmt.Printf("Usage example host\n")
		return
	}
	host := os.Args[1]
	loss, avgTime, err := ping2.Ping(host, 5)
	if err != nil{
		fmt.Println(err)
		return
	}
	fmt.Printf("loss:%v, avgTime:%v\n",loss,avgTime)
}