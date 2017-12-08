package ping2

import (
"net"
"time"
"errors"
"fmt"
"golang.org/x/net/icmp"
"golang.org/x/net/ipv4"
"math/rand"
"os"
"sort"
)


type Reply struct {
	Time float64
	TTL  uint8
}

var data = []byte("abcdefghijklmnopqrstuvwabcdefghi")

func lookup(host string) (string, error) {
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

func marshalMsg(req int, data []byte) ([]byte, error) {
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

func dial(host string) (net.Conn, error) {
	address, err := lookup(host)
	if err != nil {
		return nil, err
	}
	return net.Dial("ip4:icmp", address)
}

func sendPingMsg(host string) (reply Reply, err error) {
	conn, err := dial(host)
	if err != nil {
		return
	}
	conn.SetDeadline(time.Now().Add(2 * time.Second))

	wb, err := marshalMsg(8, data)
	if err != nil {
		return
	}

	start := time.Now()
	_, err = conn.Write(wb)
	if err != nil {
		return
	}
	rb := make([]byte,1500)
	var n int
	n, err = conn.Read(rb)
	if err != nil {
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

	//1 represent protocol ICMPv4
	rm, err := icmp.ParseMessage(1, rb[:n])
	if err != nil {
		return
	}

	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		t := float64(duration) / float64(time.Millisecond)
		reply = Reply{t, ttl}
	case ipv4.ICMPTypeDestinationUnreachable:
		err = errors.New("Destination Unreachable")
	default:
		err = fmt.Errorf("Not ICMPTypeEchoReply %v", rm)

	}
	return

}

// count must > 2
func Ping(ip string, count int) (loss, timeout float64, err error) {

	if count <= 2 {
		err = errors.New("count must bigger than 2")
		return
	}

	var (
		countLoss int
		avgTime   []float64
		sumTime   float64
	)
	start := time.Now()

	for i := 0; i < count; i++ {
		r, err := sendPingMsg(ip)
		if err != nil {
			if opt, ok := err.(*net.OpError); ok && opt.Timeout() {
				fmt.Printf("From %s reply TimeOut\n", ip)
			} else {
				fmt.Printf("From %s reply:%s\n", ip, err.Error())
			}
			countLoss++
		} else {
			fmt.Printf("From %s reply, time=%f ms, ttl=%d\n", ip, r.Time, r.TTL)
			avgTime = append(avgTime, r.Time)
		}
		time.Sleep(time.Second)
	}
	loss = float64(countLoss/count) * 100

	if len(avgTime) == 0{
		duration := time.Now().Sub(start)
		timeout = float64(duration) / float64(time.Millisecond)
		return
	}

	sort.Float64s(avgTime)
	if count-countLoss <= 2 {
		for _, v := range avgTime {
			sumTime += v
		}
		timeout = sumTime / float64(count-countLoss)
	} else {
		for i := 1; i < count-countLoss-1; i++ {
			sumTime += avgTime[i]
		}
		timeout = sumTime / float64(count-countLoss-2)
	}
	return
}
