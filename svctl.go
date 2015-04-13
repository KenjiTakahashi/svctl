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

// svctl is an interactive runit controller.
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/peterh/liner"
)

// status Represents current status of a single process.
// Note that it gather all information during construction,
// so it is generally meant to be short lived.
type status struct {
	name string
	err  error

	Offsets []int

	sv       []byte
	svStatus string
	svPid    uint
	svTime   uint64
}

// newStatus Creates new status representation for given directory and name.
func newStatus(dir, name string) *status {
	s := &status{Offsets: make([]int, 2), name: name}
	s.Offsets[0] = len(s.name)

	status, err := s.status(dir)
	if err != nil {
		s.err = err

		s.Offsets[1] = len("ERROR")
	} else {
		s.svStatus = svStatus(status)
		s.svPid = svPid(status)
		s.svTime = svTime(status)

		s.Offsets[1] = len(s.svStatus)
		if s.svStatus == "RUNNING" {
			s.Offsets[1] += len(fmt.Sprintf(" (pid %d)", s.svPid))
		}
	}
	s.sv = status

	return s
}

// status Reads current status from specified dir.
func (s *status) status(dir string) ([]byte, error) {
	if _, err := os.OpenFile(path.Join(dir, "supervise/ok"), os.O_WRONLY, 0600); err != nil {
		return nil, fmt.Errorf("unable to open supervise/ok")
	}

	fstatus, err := os.Open(path.Join(dir, "supervise/status"))
	if err != nil {
		return nil, fmt.Errorf("unable to open supervise/status")
	}

	b := make([]byte, 20)
	_, err = io.ReadFull(fstatus, b)
	fstatus.Close()
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("unable to read supervise/status: wrong format")
		}
		return nil, fmt.Errorf("unable to read supervise/status")
	}
	return b, nil
}

// Check Performs svCheck on status, if retrieved successfully.
func (s *status) Check(action []byte, start uint64) bool {
	if s.err != nil {
		return true
	}
	return svCheck(action, s.sv, start)
}

// CheckControl Performs svCheckControl on status.
func (s *status) CheckControl(action []byte) bool {
	return svCheckControl(action, s.sv)
}

// String Returns nicely stringified version of the status.
//
// s.Offsets can be set from the outside to make indentation uniform
// among multiple statuses.
func (s *status) String() string {
	var status bytes.Buffer
	fmt.Fprintf(&status, "%-[1]*s", s.Offsets[0]+3, s.name)
	if s.err != nil {
		fmt.Fprintf(&status, "%-[1]*s%s", s.Offsets[1]+3, "ERROR", s.err)
		return status.String()
	}
	fmt.Fprintf(&status, s.svStatus)
	if s.svStatus == "RUNNING" {
		fmt.Fprintf(&status, " (pid %d)", s.svPid)
	}
	fmt.Fprintf(
		&status, "%-[1]*s%ds",
		s.Offsets[1]+3-status.Len()+s.Offsets[0]+3, "", svNow()-s.svTime,
	)
	return status.String()
}

// Errored Returns whether status retrieval ended with error or not.
func (s *status) Errored() bool {
	return s.err != nil
}

// ctl Represents main svctl entry point.
type ctl struct {
	line    *liner.State
	basedir string
}

// newCtl Creates new ctl instance.
// Initializes input prompt, reads history, reads $SVDIR.
func newCtl() *ctl {
	c := &ctl{line: liner.NewLiner()}

	fn, _ := xdg.DataFile("svctl/hist")
	if f, err := os.Open(fn); err == nil {
		c.line.ReadHistory(f)
		f.Close()
	}
	c.basedir = os.Getenv("SVDIR")
	if c.basedir == "" {
		c.basedir = "/service"
	}

	c.line.SetTabCompletionStyle(liner.TabPrints)
	c.line.SetCompleter(func(l string) []string {
		s := strings.Split(l, " ")
		if len(s) <= 1 {
			if len(s) == 0 {
				return cmdMatchName("")
			}
			return cmdMatchName(s[0])
		}
		services := c.Services(fmt.Sprintf("%s*", s[len(s)-1]), true)
		compl := make([]string, len(services))
		for i, service := range services {
			compl[i] = fmt.Sprintf(
				"%s %s ",
				strings.Join(s[:len(s)-1], " "), c.serviceName(service),
			)
		}
		return compl
	})

	return c
}

// Close Closes input prompt, saves history to file.
func (c *ctl) Close() {
	fn, _ := xdg.DataFile("svctl/hist")
	if f, err := os.Create(fn); err == nil {
		if n, err := c.line.WriteHistory(f); err != nil {
			log.Printf("error writing history file: %s, lines written: %d\n", err, n)
		}
	} else {
		log.Printf("error opening history file: %s\n", err)
	}
	c.line.Close()
}

// serviceName Returns name of the service, i.e. directory chain relative to current base.
func (c *ctl) serviceName(dir string) string {
	if name, err := filepath.Rel(c.basedir, dir); err == nil {
		return name
	}
	return dir
}

// Services Returns paths to all services matching pattern.
func (c *ctl) Services(pattern string, toLog bool) []string {
	if len(pattern) < len(c.basedir) || pattern[:len(c.basedir)] != c.basedir {
		pattern = path.Join(c.basedir, pattern)
	}
	files, err := filepath.Glob(pattern)
	if err != nil {
		log.Printf("error getting services list: %s\n", err)
	}
	if toLog {
		logs, err := filepath.Glob(path.Join(pattern, "log"))
		if err != nil {
			log.Printf("error getting logs list: %s\n", err)
		} else {
			files = append(files, logs...)
			sort.Strings(files)
		}
	}
	return files
}

// Status Prints all statuses matching id and optionally their log process statuses.
func (c *ctl) Status(id string, toLog bool) {
	// TODO: normally (up|down) and stuff?
	statuses := []*status{}
	for _, dir := range c.Services(id, toLog) {
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			continue
		}

		status := newStatus(dir, c.serviceName(dir))
		statuses = append(statuses, status)

		for i, offset := range status.Offsets {
			if statuses[0].Offsets[i] < offset {
				statuses[0].Offsets[i] = offset
			}
		}
	}
	for _, status := range statuses {
		status.Offsets = statuses[0].Offsets
		fmt.Println(status)
	}
}

// control Sends action byte to service.
func (c *ctl) control(action []byte, service string) error {
	f, err := os.OpenFile(
		path.Join(service, "supervise/control"), os.O_WRONLY, 0600,
	)
	if err != nil {
		return fmt.Errorf("%s: unable to open supervise/control", path.Base(service))
	}
	defer f.Close()
	if _, err := f.Write(action); err != nil {
		return fmt.Errorf("%s: unable to write to supervise/control", path.Base(service))
	}
	return nil
}

// ctl Delegates a single action for single service.
func (c *ctl) ctl(action []byte, service string, start uint64, wg *sync.WaitGroup) {
	defer wg.Done()

	status := newStatus(service, c.serviceName(service))
	if status.Errored() {
		return
	}
	if status.CheckControl(action) {
		if err := c.control(action, service); err != nil {
			fmt.Println(err)
			return
		}
	}

	timeout := time.After(7 * time.Second)
	tick := time.Tick(100 * time.Millisecond)
	for {
		select {
		case <-timeout:
			fmt.Printf("TIMEOUT: ")
			c.Status(service, false)
			return
		case <-tick:
			status := newStatus(service, c.serviceName(service))
			if status.Check(action, start) {
				fmt.Println(status)
				return
			}
		}
	}
}

// Ctl Handles command supplied by user.
//
// Depending on the command, it might just exit, print help or propagate
// command to cmds to delegate action to runsv.
//
// If more than one service was specified with the command,
// actions are delegated asynchronically.
func (c *ctl) Ctl(cmd string) bool {
	c.line.AppendHistory(cmd)
	start := svNow()
	params := strings.Split(cmd, " ")
	var action []byte
	switch params[0] {
	case "e", "exit":
		return true
	case "s", "status":
		if len(params) == 1 {
			c.Status("*", true)
		} else {
			for _, dir := range params[1:] {
				c.Status(dir, true)
			}
		}
		return false
	case "?", "help":
		if len(params) == 1 {
			for _, cmd := range cmdAll() {
				fmt.Println(cmd.Help())
			}
			return false
		}
		for _, param := range params[1:] {
			cmd := cmdMatch(param)
			if cmd == nil {
				fmt.Printf("%s: unable to find action\n", param)
			} else {
				fmt.Println(cmd.Help())
			}
		}
		return false
	default:
		cmd := cmdMatch(params[0])
		if cmd == nil {
			fmt.Printf("%s: unable to find action\n", params[0])
			return false
		}
		action = cmd.Action()
	}

	if len(params) == 1 {
		params = append(params, "*")
	}
	var wg sync.WaitGroup
	for _, param := range params[1:] {
		if param == "" {
			continue
		}
		for _, service := range c.Services(param, false) {
			wg.Add(1)
			go c.ctl(action, service, start, &wg)
		}
	}
	wg.Wait()

	return false
}

// Run Performs one tick of a input prompt event loop.
// If this function returns true, the outside loop should terminate.
func (c *ctl) Run() bool {
	cmd, err := c.line.Prompt("svctl> ")
	if err == io.EOF {
		fmt.Println()
		return true
	} else if err != nil {
		log.Printf("error reading prompt contents: %s\n", err)
		return false
	}
	return c.Ctl(cmd)
}

// main Creates svctl entry point, prints all processes statuses and launches event loop.
func main() {
	ctl := newCtl()
	defer ctl.Close()
	ctl.Status("*", true)
	for !ctl.Run() {
	}
}
