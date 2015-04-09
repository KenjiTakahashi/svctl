package main

import (
	"fmt"
	"strings"
)

func contains(slice []string, str string) bool {
	for _, elem := range slice {
		if elem == str {
			return true
		}
	}
	return false
}

type cmd interface {
	Action() []byte
	Help() string
	Names() []string
}

type cmdMatcher interface {
	Match(name string) bool
}

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
	return []string{"restart"}
}

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
	return []byte(c.action[0:1])
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

func cmdAll() []cmd {
	return []cmd{
		&cmdUp{},
		&cmdDown{},
		&cmdRestart{},
		&cmdSignal{},
	}
}

func cmdMatch(name string) cmd {
	for _, cmd := range cmdAll() {
		m, ok := cmd.(cmdMatcher)
		if (ok && m.Match(name)) || contains(cmd.Names(), name) || string(cmd.Action()) == name {
			return cmd
		}
	}
	return nil
}

func cmdMatchName(partialName string) []string {
	res := []string{}
	for _, cmd := range cmdAll() {
		for _, name := range cmd.Names() {
			if strings.HasPrefix(name, partialName) {
				res = append(res, fmt.Sprintf("%s ", name))
			}
		}
	}
	return res
}
