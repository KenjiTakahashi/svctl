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
	"testing"

	"github.com/peterh/liner"
)

func TestHelp(t *testing.T) {
	defs := []struct {
		action string
		nlines int
	}{
		{"", 38},
		{"up", 2},
		{"down hup", 4},
		{"help", 2},
		{"help exit", 3},
	}

	stdout := &stdout{}
	svctl := ctl{line: liner.NewLiner(), stdout: stdout}

	for _, def := range defs {
		svctl.Ctl(fmt.Sprintf("help %s", def.action))

		n := stdout.Len()
		if n != def.nlines {
			t.Errorf("ERROR IN NLINES: `%d` != `%d` for `%s`", n, def.nlines, def.action)
		}

		stdout.Clear()
	}

	svctl.Ctl("help wrongaction")
	output := stdout.ReadString()
	expected := "wrongaction: unable to find action"
	if output != expected {
		t.Errorf("ERROR IN STATUS: `%s` != `%s` for `%s`", output, expected, "wrongaction")
	}

	svctl.line.Close()
}
