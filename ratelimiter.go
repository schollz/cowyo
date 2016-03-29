package main

import (
	"fmt"
	"time"
)

var bannedIPs []string

func init() {
	go clearBannedIPs()
}

func clearBannedIPs() {
	for {
		fmt.Println("CLEARING IPS!!")
		bannedIPs = []string{}
		time.Sleep(3 * time.Minute)
	}
}

func isIPBanned(ip string) bool {
	if stringInSlice(ip, bannedIPs) {
		return true
	} else {
		bannedIPs = append(bannedIPs, ip)
		return false
	}
}
