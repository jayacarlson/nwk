package nwk

import (
	"testing"

	"github.com/jayacarlson/dbg"
)

func TestIP(*testing.T) {
	ip, _ := FindMyIP4Addr("")
	dbg.Info("My IPAddr: %s", ip)
}

func TestIPPort(*testing.T) {
	ip, _ := FindMyIP4Addr(":1234")
	dbg.Info("My IPAddr: %s", ip)
}

func TestIPLoop(*testing.T) {
	ip, _ := FindMyIP4Addr("127")
	dbg.Info("My IPAddr: %s", ip)
}

func TestIPLoopPort(*testing.T) {
	ip, _ := FindMyIP4Addr("127:1234")
	dbg.Info("My IPAddr: %s", ip)
}
