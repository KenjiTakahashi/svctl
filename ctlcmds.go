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

import "strings"

// ctlCmd Defines methods common for svctl meta-commands, i.e. ones
// that are not sent to runit, but executed locally.
type ctlCmd interface {
	Run(ctl *ctl, params []string) bool
}

// ctlCmdStatus Defines the "status" action.
type ctlCmdStatus struct{}

func (c *ctlCmdStatus) Action() []byte {
	return []byte{'s'}
}

func (c *ctlCmdStatus) Help() string {
	return strings.TrimSpace(`
status [NAMES...]   Shows status(es) of service(s) with matching NAMES.
                    When invoked without NAMES, shows statuses of all processes.
                    NAMES support globing with '*' and '?'.
	`)
}

func (c *ctlCmdStatus) Names() []string {
	return []string{"status"}
}

func (c *ctlCmdStatus) Run(ctl *ctl, params []string) bool {
	if len(params) == 1 {
		ctl.Status("*", true)
	} else {
		for _, dir := range params[1:] {
			if dir == "" {
				continue
			}
			ctl.Status(dir, true)
		}
	}
	return false
}

// ctlCmdHelp Defines the "help" action.
// Note: Acronym is '?' here, because 'h' is taken by "hup".
type ctlCmdHelp struct{}

func (c *ctlCmdHelp) Action() []byte {
	return []byte{'?'}
}

func (c *ctlCmdHelp) Help() string {
	return strings.TrimSpace(`
help [CMDS...]   Shows help message(s) about CMDS.
                 When invoked without CMDS, shows available CMDS.
	`)
}

func (c *ctlCmdHelp) Names() []string {
	return []string{"help"}
}

func (c *ctlCmdHelp) Run(ctl *ctl, params []string) bool {
	if len(params) == 1 {
		for _, cmd := range cmdAll() {
			match, ok := cmd.(cmdMatcher)
			if !ok {
				ctl.println(cmd.Help())
				continue
			}
			for _, name := range cmd.Names() {
				match.Match(name)
				ctl.println(cmd.Help())
			}
		}
		return false
	}
	for _, param := range params[1:] {
		cmd := cmdMatch(param)
		if cmd == nil {
			ctl.printf("%s: unable to find action\n", param)
		} else {
			ctl.println(cmd.Help())
		}
	}
	return false
}

// ctlCmdExit Defines the "exit" action.
type ctlCmdExit struct{}

func (c *ctlCmdExit) Action() []byte {
	return []byte{'e'}
}

func (c *ctlCmdExit) Help() string {
	return "exit   Exists svctl."
}

func (c *ctlCmdExit) Names() []string {
	return []string{"exit"}
}

func (c *ctlCmdExit) Run(ctl *ctl, params []string) bool {
	return true
}
