// Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reserved.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
)

type specialOnlyString string

func (s specialOnlyString) Generate(rand *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(specialOnlyString(generateStringWithChars(
		rand,
		size,
		specialChars, whitespaceChars,
	)))
}

type specialWeakString string

func (s specialWeakString) Generate(rand *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(specialWeakString(generateStringWithChars(
		rand,
		size,
		specialChars, whitespaceChars, strongChars,
	)))
}

type specialStrongString string

func (s specialStrongString) Generate(rand *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(specialStrongString(generateStringWithChars(
		rand,
		size,
		specialChars, whitespaceChars, weakChars,
	)))
}

type specialStrongWeakString string

func (s specialStrongWeakString) Generate(rand *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(specialStrongWeakString(generateStringWithChars(
		rand,
		size,
		specialChars, whitespaceChars, strongChars, weakChars,
	)))
}

type noSpecialString string

func (s noSpecialString) Generate(rand *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(noSpecialString(generateStringWithChars(
		rand,
		size,
	)))
}

func checkPrefixSuffix(in, pfx, sfx string) bool {
	return strings.HasPrefix(in, pfx) &&
		strings.HasSuffix(in, sfx)
}

func generateStringWithChars(
	rand *rand.Rand,
	size int,
	atLeastOneOfEach ...string,
) string {
	const az = "abcdefghijklmnopqrstuvwxyz"
	const num = "1234567890"
	set := az + num
	for _, atLeastOneOf := range atLeastOneOfEach {
		set += atLeastOneOf
	}

	out := make([]byte, size+len(atLeastOneOfEach))
	for i := 0; i < size; i++ {
		ch := set[rand.Intn(len(set))]
		out[i] = ch
	}
	for i, atLeastOneOf := range atLeastOneOfEach {
		ch := atLeastOneOf[rand.Intn(len(atLeastOneOf))]
		out[size+i] = ch
	}

	return string(out)
}

func TestQuote(t *testing.T) {
	t.Run("no-special-characters", func(t *testing.T) {
		err := quick.Check(func(s noSpecialString) bool {
			return checkPrefixSuffix(quote(string(s)), "", "")
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("special-characters", func(t *testing.T) {
		err := quick.Check(func(s specialOnlyString) bool {
			return checkPrefixSuffix(quote(string(s)), "'", "'")
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("special-characters/weak", func(t *testing.T) {
		err := quick.Check(func(s specialWeakString) bool {
			return checkPrefixSuffix(quote(string(s)), "'", "'")
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("special-characters/strong", func(t *testing.T) {
		err := quick.Check(func(s specialStrongString) bool {
			return checkPrefixSuffix(quote(string(s)), "\"", "\"")
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("special-characters/strong+weak", func(t *testing.T) {
		err := quick.Check(func(s specialStrongWeakString) bool {
			return checkPrefixSuffix(quote(string(s)), "$'", "'")
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
	})
}
