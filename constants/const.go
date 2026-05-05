package constants

import (
	"time"
)

// DefaultGameserverNet contains the default list of game server IPs.
var DefaultGameserverNet = []string{
	"34.218.42.114", // Ch1 - 11020
	"50.112.234.180", // Ch2 - 11021
	"52.32.149.152", // Ch3 - 11022
	"52.34.200.191", // Ch4 - 11020
	"35.160.179.201", // Ch5 - 11021
	"52.27.21.224", // Ch6 - 11022
	"52.41.162.90", // Ch7 - 11023
	"54.70.187.83", // Ch8 - 11020
	"52.39.64.186", // Ch9 - 11021
	"52.11.161.60", // Ch10 - 11022
	"54.213.53.13", // Ch11 - 11022
	"44.234.73.29", // Ch12 - 11022
	"35.162.195.251", // Ch13 - 11022
	"44.250.19.242", // Ch13 - 11022
	"44.253.9.16", // Ch13 - 11022
	"44.230.175.95", // Ch13 - 11022
	"52.12.219.11", // hCh - 11023
}

// DefaultGameserverPort contains the default list of game server ports.
var DefaultGameserverPort = []string{"11020", "11021", "11022", "11023", "11024", "11025"}

// This variable is no longer used to build the filter dynamically, but can be kept for reference
// or removed. The dynamic building now happens in main.go.
var PCAP_GAMESERVER_FILTER = ""

var SERVER_START_AT = time.Now().Unix()
