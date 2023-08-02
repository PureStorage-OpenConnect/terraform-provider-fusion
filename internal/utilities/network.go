/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package utilities

import (
	"net/netip"
	"regexp"
)

var (
	ipRegex   = regexp.MustCompile(`^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}$`)
	cidrRegex = regexp.MustCompile(`^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}\/(?:[8-9]|[1-2]\d|3[0-2])$`)
)

func IsValidAddress(addr string) bool {
	return ipRegex.MatchString(addr)
}

func IsValidCidr(prefix string) bool {
	return cidrRegex.MatchString(prefix)
}

func IsAddressInPrefix(addr, prefix string) bool {
	if !IsValidAddress(addr) || !IsValidCidr(prefix) {
		return false
	}

	parsedAddr, _ := netip.ParseAddr(addr)
	parsedPrefix, _ := netip.ParsePrefix(prefix)

	return parsedPrefix.Contains(parsedAddr)
}
