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
package ping

import (
	"errors"
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"math"
	"math/rand"
	"net"
	"os"
	"time"
)

// A Header represents an IPv4 header.
// but only part of IP Header
type IPHeader struct {
	TOS int // type-of-service
	TTL int // time-to-live
}

type Params struct {
	Persist bool
	Sip     string
	Count   int
	Dip string
}

type Reply struct {
	Time  int64
	TTL   uint8
	Error error
}

type ping struct {
	Conn net.Conn
	Data []byte
	*IPHeader
	*Params
}

func NewIPHeader(tos, ttl int) *IPHeader {
	if tos < -1 {
		tos = 0
	}
	if tos > 255 {
		tos = 255
	}
	if ttl < -1 {
		ttl = 0
	}
	if ttl > 255 {
		ttl = 255
	}
	return &IPHeader{TOS: tos, TTL: ttl}
}

func NewParams(persist bool, sip ,dip string, count int) *Params {
	dip, err := Lookup(dip)
	if err != nil {
		fmt.Println("Lookup error:",err)
	}
	return &Params{Persist: persist, Sip: sip, Dip: dip, Count: count}
}

func NewPing(req int, data []byte, ipheader *IPHeader, params *Params) (*ping, error) {
	wb, err := MarshalMsg(req, data)
	if err != nil {
		return nil, err
	}
	return &ping{Data: wb, IPHeader: ipheader, Params: params}, nil
}

// LookupHost looks up the given host using the local resolver.
// It returns an array of that host's addresses and error.
func Lookup(host string) (string, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}
	if len(addrs) < 1 {
		return "", errors.New("unknown host")
	}
	rd := rand.New(rand.NewSource(time.Now().UnixNano()))
	return addrs[rd.Intn(len(addrs))], nil
}

//return the Marshal Message
func MarshalMsg(req int, data []byte) ([]byte, error) {
	xid, xseq := os.Getpid()&0xffff, req
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: xid, Seq: xseq,
			Data: data,
		},
	}
	return wm.Marshal(nil)
}

//new a Conn and save the Conn into ping.Conn
func (self *ping) Dail() (err error) {
	laddr := net.IPAddr{IP: net.ParseIP(self.Sip)}
	raddr := net.IPAddr{IP:net.ParseIP(self.Dip)}

	self.Conn, err = net.DialIP("ip4:icmp",&laddr,&raddr)
	//self.Conn, err = net.Dial("ip4:icmp", self.Addr)
	if err != nil {
		return err
	}
	return nil
}

// SetDeadline sets the read and write deadlines associated
// with the connection.
func (self *ping) SetDeadline(timeout int) error {
	return self.Conn.SetDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
}

func (self *ping) SetTOS() {
	p := ipv4.NewConn(self.Conn)
	if err := p.SetTOS(self.TOS); err != nil {
		// DSCP AF11
		fmt.Println("SetTOS:", err)
	}

}

func (self *ping) SetTTL() {
	p := ipv4.NewConn(self.Conn)
	if err := p.SetTTL(self.TTL); err != nil {
		fmt.Println("SetTTL error:", err)
	}
}

func (self *ping) Close() error {
	return self.Conn.Close()
}

func (self *ping) Ping() {
	if err := self.Dail(); err != nil {
		fmt.Println("Not found remote host")
		return
	}
	fmt.Println("Start ping from ", self.Conn.LocalAddr())
	//TODO: set read and write timeout
	self.SetDeadline(10)

	if self.Persist == true {
		self.Count = math.MaxInt32
	}
	for i := 0; i < self.Count; i++ {
		r := self.sendPingMsg()
		if r.Error != nil {
			if opt, ok := r.Error.(*net.OpError); ok && opt.Timeout() {
				fmt.Printf("From %s reply: TimeOut\n", self.Dip)
				if err := self.Dail(); err != nil {
					fmt.Println("Not found remote host")
					return
				}
			} else {
				fmt.Printf("From %s reply: %s\n", self.Dip, r.Error)
			}
		} else {
			fmt.Printf("From %s reply: time=%d ttl=%d\n", self.Dip, r.Time, r.TTL)
		}
		//TODO: send packet interval
		time.Sleep(1e9)
	}
}

func (self *ping) sendPingMsg() (reply Reply) {
	start := time.Now()

	self.SetTOS()
	self.SetTTL()

	if _, reply.Error = self.Conn.Write(self.Data); reply.Error != nil {
		return
	}

	rb := make([]byte, 1500)
	var n int
	n, reply.Error = self.Conn.Read(rb)
	if reply.Error != nil {
		return
	}

	duration := time.Now().Sub(start)
	ttl := uint8(rb[8])

	rb = func(b []byte) []byte {
		if len(b) < 20 {
			return b
		}
		hdrlen := int(b[0]&0x0f) << 2
		return b[hdrlen:]
	}(rb)

	var rm *icmp.Message

	//1 represent protocol ICMPv4
	rm, reply.Error = icmp.ParseMessage(1, rb[:n])
	if reply.Error != nil {
		return
	}

	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		t := int64(duration / time.Millisecond)
		reply = Reply{t, ttl, nil}
	case ipv4.ICMPTypeDestinationUnreachable:
		reply.Error = errors.New("Destination Unreachable")
	default:
		reply.Error = fmt.Errorf("Not ICMPTypeEchoReply %v", rm)
	}
	return
}
