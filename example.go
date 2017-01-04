// Copyright 2017 nicle-lin. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"flag"
	"fmt"
	"github.com/nicle-lin/ping/ping"
	"os"
	"runtime"
)


var usage = `Usage: ping target_name(target_ip) [options]
options:
   -v: Set IP packet Tos
   -t: Using this option will ping the target until you force it to stop using Ctrl-C
   -n: This option sets the number of ICMP Echo Request messages to send.
       If you execute the ping command without this option, four requests will be sent.
   -i: This option sets the Time to Live (TTL) value, the maximum of which is 255.
   -S: Use this option to specify the source address.
`

var (
	tos     = flag.Int("v", 0, "TOS")
	persist = flag.Bool("t", false, "")
	count   = flag.Int("n", 4, "count")
	ttl     = flag.Int("i", 128, "TTL")
	sip     = flag.String("S", "0.0.0.0", "Source IP")
)

var Data = []byte("abcdefghijklmnopqrstuvwabcdefghi")


func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(usage, runtime.NumCPU()))
	}
	flag.Parse()

	if flag.NArg() < 1 {
		usageAndExit("")
	}

	host := flag.Args()[0]
	ipHeader := ping.NewIPHeader(*tos, *ttl)
	params := ping.NewParams(*persist, *sip,host, *count)

	ping, err := ping.NewPing(8, Data, ipHeader, params)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer ping.Close()
	ping.Ping()

}

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}
