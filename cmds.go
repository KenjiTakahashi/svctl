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
	return ""
}

func (c *cmdUp) Names() []string {
	return []string{"up", "start"}
}

type cmdDown struct{}

func (c *cmdDown) Action() []byte {
	return []byte("d")
}

func (c *cmdDown) Help() string {
	return ""
}

func (c *cmdDown) Names() []string {
	return []string{"down", "stop"}
}

type cmdRestart struct{}

func (c *cmdRestart) Action() []byte {
	return []byte("tcu")
}

func (c *cmdRestart) Help() string {
	return ""
}

func (c *cmdRestart) Names() []string {
	return []string{"restart"}
}

type cmdSignal struct {
	action []byte
}

func (c *cmdSignal) Action() []byte {
	return c.action
}

func (c *cmdSignal) Help() string {
	return ""
}

func (c *cmdSignal) Names() []string {
	return []string{
		"pause", "cont", "hup", "alarm", "interrupt",
		"quit", "1", "2", "term", "kill",
	}
}

func (c *cmdSignal) Match(name string) bool {
	for _, s := range c.Names() {
		if name == s || name == s[0:1] {
			c.action = []byte(name[0:1])
			return true
		}
		if name == "reload" {
			c.action = []byte("h")
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
