package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/peterh/liner"
)

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

const basedir = "/etc/service" // TODO: Make configurable

type ctl struct {
	line *liner.State
}

func newCtl() *ctl {
	return &ctl{
		line: liner.NewLiner(),
	}
}

func (c *ctl) Close() {
	c.line.Close()
}

func (c *ctl) Services(pattern string) []string {
	if len(pattern) < len(basedir) || pattern[:len(basedir)] != basedir {
		pattern = path.Join(basedir, pattern)
	}
	files, err := filepath.Glob(pattern)
	fatal(err)
	return files
}

func (c *ctl) printStatus(dir string) {
	if status, err := c.status(dir); err != nil {
		fmt.Printf(": %s", err)
	} else {
		sv := svStatus(status)
		fmt.Printf(": %s", sv)
		if sv == "RUNNING" {
			fmt.Printf(" (pid %d)", svPid(status))
		}
		fmt.Printf(", %ds", svNow()-svTime(status))
	}
}

func (c *ctl) status(dir string) ([]byte, error) {
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

func (c *ctl) Status(id string, log bool) {
	// TODO: normally (up|down) and stuff?
	for _, dir := range c.Services(id) {
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			continue
		}

		fmt.Print(path.Base(dir))
		c.printStatus(dir)

		if log {
			logdir := path.Join(dir, "log")
			if _, err := os.Stat(logdir); os.IsNotExist(err) {
				fmt.Println()
				continue
			}

			fmt.Print(" ;log")
			c.printStatus(logdir)
		}

		fmt.Println()
	}
}

func (c *ctl) Ctl(cmd string) bool {
	params := strings.Split(cmd, " ")
	var action []byte
	switch params[0] {
	case "q", "quit":
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
		for _, service := range c.Services(param) {
			// TODO: Check status for not running, once w/o TERM, etc.
			f, err := os.OpenFile(
				path.Join(service, "supervise/control"), os.O_WRONLY, 0600,
			)
			fatal(err)
			_, err = f.Write(action)
			fatal(err)
			f.Close()

			wg.Add(1)
			go func(service string) {
				defer wg.Done()
				// TODO: Better waiting algorithm
				time.Sleep(1 * time.Second)
				c.Status(service, false)
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
		fmt.Println(err) // TODO: Better error handling
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
