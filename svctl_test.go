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
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/peterh/liner"
)

func fatal(err error) {
	if err != nil {
		panic(err)
	}
}

type runitRunner struct {
	basedir  string
	runsvdir *exec.Cmd
	zs       map[string]int

	stdout     *bufio.Reader
	realStdout *os.File
}

func newRunitRunner() *runitRunner {
	dir, err := ioutil.TempDir("", "svctl_tests")
	fatal(err)
	cmd := exec.Command("cp", "-r", "_testdata/", dir)
	fatal(cmd.Run())

	r := &runitRunner{
		basedir: path.Join(dir, "_testdata"),
		zs:      map[string]int{"r0": 0, "r1": 0, "o": 0},
	}

	stdr, stdw, err := os.Pipe()
	fatal(err)
	r.stdout = bufio.NewReader(stdr)
	r.realStdout = os.Stdout
	os.Stdout = stdw

	r.runsvdir = exec.Command("runsvdir", "-P", r.basedir)
	fatal(r.runsvdir.Start())
	// Make sure runsvdir has enough time to scan the directories
	time.Sleep(5 * time.Second)

	return r
}

func (r *runitRunner) Close() {
	r.runsvdir.Process.Signal(syscall.SIGHUP)
	r.runsvdir.Process.Wait()
	os.RemoveAll(path.Dir(r.basedir))
	os.Stdout.Close()
	os.Stdout = r.realStdout
}

func (r *runitRunner) Assert(t *testing.T, cmd *cmdDef) {
	for _, service := range cmd.services {
		stdout, err := r.stdout.ReadString('\n')
		if err != nil {
			t.Errorf("ERROR READING `stdout`: `%s`", err)
			continue
		}
		pieces := strings.Split(stdout, "   ")
		if !contains(cmd.services, pieces[0]) {
			t.Errorf(
				"ERROR IN STATUS: service `%s` != `%s` for %s:%s",
				pieces[0], service, cmd.cmd, service,
			)
			continue
		}
		statusPiece := strings.SplitN(strings.TrimSpace(pieces[1]), " ", 2)[0]
		if statusPiece != cmd.status {
			t.Errorf(
				"ERROR IN STATUS: status `%s` != `%s` for %s:%s",
				statusPiece, cmd.status, cmd.cmd, service,
			)
		}
	}

	noNewZ := func(service string, z int) {
		signal, err := ioutil.ReadFile(path.Join(
			r.basedir, service, fmt.Sprintf("z.%d", z+1),
		))
		if os.IsNotExist(err) { // This is what should happen
			return
		}
		t.Errorf(
			"ERROR IN z FILE: `z.%d` (with `%s`) should not exist for %s:%s",
			z+1, signal, service, cmd.cmd,
		)
	}
	for service, z := range r.zs {
		if cmd.z != 0 { // Service(s) were supposed to get new signal
			if contains(cmd.services, service) { // and we are at one of these services
				r.zs[service] = cmd.z
				signal, err := ioutil.ReadFile(path.Join(
					r.basedir, service, fmt.Sprintf("z.%d", cmd.z),
				))
				if err != nil {
					t.Errorf("ERROR IN z FILE: %s", err)
					continue
				}
				expected := cmd.cmd[0]
				// Corner cases, not worth extending cmdDef
				if cmd.cmd == "reload" {
					expected = 'h'
				}
				if expected == 'd' || expected == 'r' {
					expected = 't'
				}
				if signal[0] != expected {
					t.Errorf(
						"ERROR IN z FILE: `%c` != `%c` for %s:%s",
						signal[0], expected, service, cmd.cmd,
					)
				}
			} else { // or not
				noNewZ(service, z)
			}
		} else { // Service(s) were NOT supposed to get new signal
			noNewZ(service, z)
		}
	}
}

func (r *runitRunner) AssertError(t *testing.T, msg string) {
	stdout, err := r.stdout.ReadString('\n')
	if err != nil {
		t.Errorf("ERROR READING `stdout`: `%s`", err)
		return
	}
	stdout = stdout[:len(stdout)-1]
	if stdout != msg {
		t.Errorf("ERROR IN STATUS: `%s` != `%s`", stdout, msg)
	}
}

type cmdDef struct {
	cmd      string
	services []string
	status   string
	z        int
}

func TestCmd(t *testing.T) {
	runit := newRunitRunner()
	svctl := ctl{basedir: runit.basedir, line: liner.NewLiner()}

	// Tests for correct usage.
	cmds := []cmdDef{
		{"u", []string{"r0", "r1"}, "RUNNING", 0},
		{"d", []string{"r0", "r1"}, "STOPPED", 1},
		{"up", []string{"r0"}, "RUNNING", 0},
		{"start", []string{"r0"}, "RUNNING", 0},
		{"down", []string{"r0"}, "STOPPED", 2},
		{"stop", []string{"r0"}, "STOPPED", 0},
		{"r", []string{"r1"}, "RUNNING", 0},
		{"restart", []string{"r1"}, "RUNNING", 2},
		{"p", []string{"r1"}, "PAUSED", 0},
		{"c", []string{"r1"}, "RUNNING", 0},
		{"pause", []string{"r0"}, "STOPPED", 0},
		{"cont", []string{"r0"}, "STOPPED", 0},
		{"h", []string{"r0"}, "STOPPED", 0},
		{"hup", []string{"r1"}, "RUNNING", 3},
		{"reload", []string{"r1"}, "RUNNING", 4},
		{"i", []string{"r0"}, "STOPPED", 0},
		{"interrupt", []string{"r1"}, "RUNNING", 5},
		{"a", []string{"r0"}, "STOPPED", 0},
		{"alarm", []string{"r1"}, "RUNNING", 6},
		{"q", []string{"r0"}, "STOPPED", 0},
		{"quit", []string{"r1"}, "RUNNING", 7},
		{"1", []string{"r1"}, "RUNNING", 8},
		{"2", []string{"r1"}, "RUNNING", 9},
		{"t", []string{"r0"}, "STOPPED", 0},
		{"term", []string{"r1"}, "RUNNING", 10},
		{"k", []string{"r0"}, "STOPPED", 0},
		{"kill", []string{"r1"}, "RUNNING", 0},
		{"o", []string{"o"}, "RUNNING", 0},
		{}, // Sleep
		{"s", []string{"o"}, "STOPPED", 0},
		{"once", []string{"r1"}, "RUNNING", 0},
		{"s", []string{"r0", "o"}, "STOPPED", 0},
	}
	for _, cmd := range cmds {
		if cmd.cmd == "" {
			time.Sleep(time.Second)
			continue
		}
		svctl.Ctl(strings.Join(append([]string{cmd.cmd}, cmd.services...), " "))
		runit.Assert(t, &cmd)
	}

	// Tests for errors.
	// Should span to other actions no problem, so just check with `u`.
	// Incorrect action.
	svctl.Ctl("n w")
	runit.AssertError(t, "n: unable to find action")
	// Incorrect service.
	svctl.Ctl("u i")
	runit.AssertError(t, "i: unable to find service")
	// No supervise/ok error.
	svctl.Ctl("u w")
	runit.AssertError(t, "w   ERROR   unable to open supervise/ok")

	svctl.line.Close()
	runit.Close()
}
