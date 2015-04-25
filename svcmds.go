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

// cmdUp Defines the "up / start" action.
type cmdUp struct{}

func (c *cmdUp) Action() []byte {
	return []byte("u")
}

func (c *cmdUp) Help() string {
	return strings.TrimSpace(`
up NAMES...   Starts service(s) with matching NAMES.
              NAMES support globing with '*' and '?'.
	`)
}

func (c *cmdUp) Names() []string {
	return []string{"up", "start"}
}

// cmdDown Defines the "down / stop" action.
type cmdDown struct{}

func (c *cmdDown) Action() []byte {
	return []byte("d")
}

func (c *cmdDown) Help() string {
	return strings.TrimSpace(`
down NAMES...   Stops service(s) with matching NAMES.
                NAMES support globing with '*' and '?'.
	`)
}

func (c *cmdDown) Names() []string {
	return []string{"down", "stop"}
}

// cmdRestart Defines the "restart" action.
type cmdRestart struct{}

func (c *cmdRestart) Action() []byte {
	return []byte("tcu")
}

func (c *cmdRestart) Help() string {
	return strings.TrimSpace(`
restart NAMES...   Restarts service(s) with matching NAMES.
                   NAMES support globing with '*' and '?'.
                   Waits up to 7 seconds for the service to get back up, then
                   reports TIMEOUT.
	`)
}

func (c *cmdRestart) Names() []string {
	return []string{"r", "restart"}
}

// cmdOnce Defines the "once" action.
type cmdOnce struct{}

func (c *cmdOnce) Action() []byte {
	return []byte{'o'}
}

func (c *cmdOnce) Help() string {
	return strings.TrimSpace(`
once NAMES...   Starts service once and does not try to restart it if it stops.
                NAMES support globing with '*' and '?'.
	`)
}

func (c *cmdOnce) Names() []string {
	return []string{"once"}
}

// cmdSignal Defines common struct for handling all actions
// that send a single *NIX signal to the process.
type cmdSignal struct {
	action string
}

func (c *cmdSignal) Action() []byte {
	if c.action == "" {
		return nil
	}
	if c.action == "reload" {
		return []byte{'h'}
	}
	return []byte{c.action[0]}
}

func (c *cmdSignal) Help() string {
	m := map[byte]string{
		'p': "STOP", 'c': "CONT", 'h': "HUP", 'r': "HUP", 'a': "ALRM", 'i': "INT",
		'q': "QUIT", '1': "USR1", '2': "USR2", 't': "TERM", 'k': "KILL",
	}
	return fmt.Sprintf(strings.TrimSpace(`
%s NAMES...   Sends signal '%s' to service(s) with matching NAMES.
%-[3]*s            NAMES support globing with '*' and '?'.
	`), c.action, m[c.action[0]], len(c.action), "")
}

func (c *cmdSignal) Names() []string {
	return []string{
		"pause", "cont", "hup", "reload", "alarm",
		"interrupt", "quit", "1", "2", "term", "kill",
	}
}

func (c *cmdSignal) Match(name string) bool {
	for _, s := range c.Names() {
		if name == s || name == s[0:1] {
			c.action = s
			return true
		}
	}
	return false
}
