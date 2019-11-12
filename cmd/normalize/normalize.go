// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// Specifically for the Ipv6String() function:
//
// Copyright (c) 2009 The Go Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
//      notice, this list of conditions and the following disclaimer.
//
//    * Redistributions in binary form must reproduce the above copyright
//      notice, this list of conditions and the following disclaimer in the
//      documentation and/or other materials provided with the distribution.
//
//    * Neither the name of Google Inc. nor the names of its contributors
//      may be used to endorse or promote products derived from this
//      software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//
// SPDX-License-Identifier: LGPL-2.1-only and BSD-3-Clause

package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/danos/mgmterror"
)

const MAC_LEN = 6 // i.e. 48 bits is 6 bytes

func invalidValueError(msg string) error {
	err := mgmterror.NewInvalidValueApplicationError()
	err.Message = msg
	return err
}

// This is needed as the built-in ParseMAC does not parse MACs with
// leading zeros removed (e.g., a:b:c:d:e:f will not parse whereas
// 0a:0b:0c:0d:0e:0f will parse). Also, the
// ietf-yang-types:mac-address type (defined in RFC6991) requires
// leading zeros.
//
// Unfortunately our old implementation normalized without leading
// zeros so we need our own processing to be backward compatible.
func parseMac48(mac string) (net.HardwareAddr, error) {
	melem := strings.Split(mac, ":")
	if len(melem) != MAC_LEN {
		return nil, invalidValueError("Incorrect size for MAC-48")
	}
	hwaddr := make(net.HardwareAddr, MAC_LEN)
	for i, v := range melem {
		a, err := strconv.ParseUint(v, 16, 8)
		if err != nil {
			return nil, invalidValueError(err.Error())
		}
		hwaddr[i] = byte(a)
	}
	return hwaddr, nil
}

// This is a copy of the standard Go library 'net/ip.go:(IP)String() method
// that is modified to return only the latter two address forms given by
// that function, ie it does not return NIL for address length zero, and nor
// does it return an IPv4-mapped IPv6 address as dotted decimal.
func Ipv6String(p net.IP) string {

	if len(p) != net.IPv6len {
		return "?"
	}

	// Find longest run of zeros.
	e0 := -1
	e1 := -1
	for i := 0; i < net.IPv6len; i += 2 {
		j := i
		for j < net.IPv6len && p[j] == 0 && p[j+1] == 0 {
			j += 2
		}
		if j > i && j-i > e1-e0 {
			e0 = i
			e1 = j
			i = j
		}
	}
	// The symbol "::" MUST NOT be used to shorten just one 16 bit 0 field.
	if e1-e0 <= 2 {
		e0 = -1
		e1 = -1
	}

	const maxLen = len("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")
	b := make([]byte, 0, maxLen)

	// Print with possible :: in place of run of zeros
	for i := 0; i < net.IPv6len; i += 2 {
		if i == e0 {
			b = append(b, ':', ':')
			i = e1
			if i >= net.IPv6len {
				break
			}
		} else if i > 0 {
			b = append(b, ':')
		}
		value := (uint32(p[i]) << 8) | uint32(p[i+1])
		b = append(b, []byte(fmt.Sprintf("%x", value))...)
	}
	return string(b)
}

func NormalizeMac(token string) string {

	if hw, err := net.ParseMAC(token); err == nil {
		return hw.String()
	}
	if hw, err := parseMac48(token); err == nil {
		return hw.String()
	}

	return token
}

func NormalizeNumber(token string) string {

	number, err := strconv.ParseUint(token, 10, 64)
	if err != nil {
		return token
	}
	return fmt.Sprintf("%d", number)
}

func NormalizeIPv4(token string) string {

	if strings.Contains(token, ":") {
		return token
	}

	if ip := net.ParseIP(token); ip != nil {
		return ip.String()
	}

	if i := strings.Index(token, "/"); i > -1 {
		addr, mask := token[:i], token[i+1:]
		return NormalizeIPv4(addr) + "/" + NormalizeNumber(mask)
	}

	return token
}

func normNeg(token string, norm func(string) string) string {
	if token[0] != '!' {
		return token
	}
	return "!" + norm(token[1:])
}

func NormalizeNegIPv4(token string) string {
	return normNeg(token, NormalizeIPv4)
}

func NormalizeIPv4prefix(token string) string {

	if strings.Contains(token, ":") {
		return token
	}

	i := strings.Index(token, "/")
	if i <= -1 {
		return token
	}

	addr_string := token[:i]
	mask_string := token[i+1:]

	addr := net.ParseIP(addr_string)
	if addr == nil {
		return token
	}

	maskbits, err := strconv.ParseUint(mask_string, 10, 64)
	if err != nil {
		return token
	}
	if maskbits > 32 {
		return fmt.Sprintf("%s/%d", addr.String(), maskbits)
	}

	mask := net.CIDRMask(int(maskbits), net.IPv4len*8)
	masked_addr := addr.Mask(mask)

	return fmt.Sprintf("%s/%d", masked_addr.String(), maskbits)
}

func NormalizeNegIPv4prefix(token string) string {
	return normNeg(token, NormalizeIPv4prefix)
}

func NormalizeIPv6prefix(token string) string {

	if !strings.Contains(token, ":") {
		return token
	}

	i := strings.Index(token, "/")
	if i <= -1 {
		return token
	}

	addr_string := token[:i]
	mask_string := token[i+1:]

	addr := net.ParseIP(addr_string)
	if addr == nil {
		return token
	}

	maskbits, err := strconv.ParseUint(mask_string, 10, 64)
	if err != nil {
		return token
	}
	if maskbits > 128 {
		return fmt.Sprintf("%s/%d", Ipv6String(addr), maskbits)
	}

	mask := net.CIDRMask(int(maskbits), net.IPv6len*8)
	masked_addr := addr.Mask(mask)

	return fmt.Sprintf("%s/%d", Ipv6String(masked_addr), maskbits)
}

func NormalizeNegIPv6prefix(token string) string {
	return normNeg(token, NormalizeIPv6prefix)
}

func NormalizeIPv6(token string) string {

	if !strings.Contains(token, ":") {
		return token
	}

	if ip := net.ParseIP(token); ip != nil {
		// We can't use the standard print function here because
		// by default it prints IPv4 mapped IPv6 addresses in
		// dotted notation
		return Ipv6String(ip)
	}

	if i := strings.Index(token, "/"); i > -1 {
		addr, mask := token[:i], token[i+1:]
		return NormalizeIPv6(addr) + "/" + NormalizeNumber(mask)
	}

	return token
}

func NormalizeNegIPv6(token string) string {
	return normNeg(token, NormalizeIPv6)
}

func NormalizeIP(token string) string {

	// Look for ":" first, since mapped v6 addresses can contain "."s
	if strings.Contains(token, ":") {
		return NormalizeIPv6(token)
	} else if strings.Contains(token, ".") {
		return NormalizeIPv4(token)
	}

	return token
}

func NormalizeNegIP(token string) string {

	// Look for ":" first, since mapped v6 addresses can contain "."s
	if strings.Contains(token, ":") {
		return NormalizeNegIPv6(token)
	} else if strings.Contains(token, ".") {
		return NormalizeNegIPv4(token)
	}

	return token
}

func NormalizeIPprefix(token string) string {

	// Look for ":" first, since mapped v6 addresses can contain "."s
	if strings.Contains(token, ":") {
		return NormalizeIPv6prefix(token)
	} else if strings.Contains(token, ".") {
		return NormalizeIPv4prefix(token)
	}

	return token
}

func NormalizeNegIPprefix(token string) string {

	// Look for ":" first, since mapped v6 addresses can contain "."s
	if strings.Contains(token, ":") {
		return NormalizeNegIPv6prefix(token)
	} else if strings.Contains(token, ".") {
		return NormalizeNegIPv4prefix(token)
	}

	return token
}

func NormalizeString(token string) string {

	token = NormalizeIP(token)
	token = NormalizeNegIP(token)
	token = NormalizeMac(token)

	return token
}
