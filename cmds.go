// svctl
// Copyright (C) 2015 Karol 'Kenji Takahashi' Woźniak
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
// DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE
// OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package main

import (
	"fmt"
	"strings"
)

// contains Checks whether str is in slice.
func contains(slice []string, str string) bool {
	for _, elem := range slice {
		if elem == str {
			return true
		}
	}
	return false
}

// cmd Defines methods common for all commads (aka. actions)
// available to svctl user through input prompt.
type cmd interface {
	Action() []byte
	Help() string
	Names() []string
}

// cmdMatcher Defines methods for commands that need custom matching functionality.
//
// By default a command string match if either is true:
// 1) .Match() is implemented and returns true.
// 2) it is equal to one of the strings returned by .Names()
// 3) it is equal to the byte returned by .Action()
type cmdMatcher interface {
	Match(name string) bool
}

// cmdAll Returns all available commands.
func cmdAll() []cmd {
	return []cmd{
		&cmdUp{},
		&cmdDown{},
		&cmdRestart{},
		&cmdOnce{},
		&cmdSignal{},
	}
}

// cmdMatch Searches available commands for one that matches name.
// Returns its instance if found, null otherwise.
func cmdMatch(name string) cmd {
	for _, cmd := range cmdAll() {
		m, ok := cmd.(cmdMatcher)
		if (ok && m.Match(name)) || contains(cmd.Names(), name) || string(cmd.Action()) == name {
			return cmd
		}
	}
	return nil
}

// cmdMatchName Searches for command names starting with `prefix`.
func cmdMatchName(prefix string) []string {
	res := []string{}
	for _, cmd := range cmdAll() {
		for _, name := range cmd.Names() {
			if strings.HasPrefix(name, prefix) {
				res = append(res, fmt.Sprintf("%s ", name))
			}
		}
	}
	return res
}
