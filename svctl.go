package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/peterh/liner"
)

type status struct {
	name string
	err  error

	Offsets []int

	sv       []byte
	svStatus string
	svPid    uint
	svTime   uint64
}

func newStatus(dir, name string) *status {
	s := &status{Offsets: make([]int, 2)}

	if name != "" {
		s.name = name
	} else {
		s.name = path.Base(dir)
	}
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

func (s *status) Check(action []byte, start uint64) bool {
	if s.err != nil {
		return true
	}
	return svCheck(action, s.sv, start)
}

func (s *status) CheckControl(action []byte) bool {
	return svCheckControl(action, s.sv)
}

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
	fmt.Fprintf(&status, "%-[1]*s", s.Offsets[1]+3-status.Len()+s.Offsets[0]+3, "")
	status.WriteString(fmt.Sprintf("%ds   ", svNow()-s.svTime))
	return status.String()
}

func (s *status) Errored() bool {
	return s.err != nil
}

type ctl struct {
	line    *liner.State
	basedir string
}

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
		services := c.Services(fmt.Sprintf("%s*", s[len(s)-1]))
		compl := make([]string, len(services))
		for i, service := range services {
			compl[i] = fmt.Sprintf(
				"%s %s ",
				strings.Join(s[:len(s)-1], " "), path.Base(service),
			)
		}
		return compl
	})

	return c
}

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

func (c *ctl) Services(pattern string) []string {
	if len(pattern) < len(c.basedir) || pattern[:len(c.basedir)] != c.basedir {
		pattern = path.Join(c.basedir, pattern)
	}
	files, err := filepath.Glob(pattern)
	if err != nil {
		log.Printf("error getting services list: %s\n", err)
	}
	return files
}

func (c *ctl) Status(id string, toLog bool) {
	// TODO: normally (up|down) and stuff?
	statuses := []*status{}
	for _, dir := range c.Services(id) {
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			continue
		}

		status := newStatus(dir, "")
		statuses = append(statuses, status)

		if toLog {
			logdir := path.Join(dir, "log")
			if _, err := os.Stat(logdir); !os.IsNotExist(err) {
				status = newStatus(logdir, fmt.Sprintf("%s/LOG", path.Base(dir)))
				statuses = append(statuses, status)
			}
		}

		for i, offset := range status.Offsets {
			if statuses[0].Offsets[i] < offset {
				statuses[0].Offsets[i] = offset
			}
		}
	}
	for _, status := range statuses {
		for i, offset := range statuses[0].Offsets {
			status.Offsets[i] = offset
		}
		fmt.Println(status)
	}
}

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

func (c *ctl) Ctl(cmd string) bool {
	c.line.AppendHistory(cmd)
	start := svNow()
	params := strings.Split(cmd, " ")
	var action []byte
	switch params[0] {
	// FIXME: "quit" is reserved by runit (and "exit" too)...
	// case "q", "quit":
	// 	return true
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
		for _, service := range c.Services(param) {
			status := newStatus(service, "")
			if status.Errored() {
				continue
			}
			if status.CheckControl(action) {
				if err := c.control(action, service); err != nil {
					fmt.Println(err)
					continue
				}
			}

			wg.Add(1)
			go func(service string) {
				defer wg.Done()
				timeout := time.After(7 * time.Second)
				tick := time.Tick(100 * time.Millisecond)
				for {
					select {
					case <-timeout:
						fmt.Printf("TIMEOUT: ")
						c.Status(service, false)
						return
					case <-tick:
						status := newStatus(service, "")
						if status.Check(action, start) {
							fmt.Println(status)
							return
						}
					}
				}
			}(service)
		}
	}
	wg.Wait()

	return false
}

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

func main() {
	ctl := newCtl()
	defer ctl.Close()
	ctl.Status("*", true)
	for !ctl.Run() {
	}
}
