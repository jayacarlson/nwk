package nwk

import (
	"net"
	"strings"

	"github.com/jayacarlson/dbg"
)

/*
	FindMyIP4Addr(lead string) (string,error):
		Will return the IP4 address for the given (if given) leading interface, i.e.
			"127"		would return the local loopback
			"127:5555"	also the local loopback, but with the port info appended > "127.0.0.1:5555"
			"192.168"	would return the first interface with "192.168" as part of it's addr
			""			giving an empty string will return the 1st non-loopback interface

		Returns found IP4 address with trailing :PORT if one given
*/

func FindMyIP4Addr(lead string) (string, error) {
	tail := ""
	i := strings.Split(lead, ":")
	if len(i) == 2 {
		tail = ":" + i[1]
		lead = i[0]
	}
	if len(lead) > 0 {
		dbg.ChkTruX(lead[0] != '.') // no leading "."

		if lead[len(lead)-1] != '.' { // force trailing "."
			lead += "."
		}
		dbg.ChkTruX(strings.Count(lead, ".") < 4) // check count, will have 3 if given full address
	}

	infs, _ := net.Interfaces() // range over the interfaces
	for _, i := range infs {
		addrs, err := i.Addrs()
		if dbg.ChkErr(err, "FindMyIPAddr (%s): %v", i.Name, err) {
			continue
		}
		for _, a := range addrs { // range over the addrs associated with the interface
			if strings.ContainsRune(a.String(), ':') {
				continue // skip IPv6 addresses
			}
			if "" != lead { // find interface that starts with "lead"
				if strings.HasPrefix(a.String(), lead) {
					return a.String()[:strings.Index(a.String(), "/")] + tail, nil
				}
			} else if i.Name != "lo" { // return 1st non-loopback interface
				return a.String()[:strings.Index(a.String(), "/")] + tail, nil
			}
		}
	}
	return "", Err_BadInterface
}
