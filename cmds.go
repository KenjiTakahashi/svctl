package main

type cmd interface {
	Action() []byte
	Help() string
	Match(name string) bool
}

type cmdUp struct{}

func (c *cmdUp) Action() []byte {
	return []byte("u")
}

func (c *cmdUp) Help() string {
	return ""
}

func (c *cmdUp) Match(name string) bool {
	return name == "u" || name == "up" || name == "start"
}

type cmdDown struct{}

func (c *cmdDown) Action() []byte {
	return []byte("d")
}

func (c *cmdDown) Help() string {
	return ""
}

func (c *cmdDown) Match(name string) bool {
	return name == "d" || name == "down" || name == "stop"
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

func (c *cmdSignal) Match(name string) bool {
	sigs := []string{
		"pause", "cont", "hup", "alarm", "interrupt",
		"quit", "1", "2", "term", "kill",
	}
	for _, s := range sigs {
		if name == s || name == s[0:1] {
			c.action = []byte(name[0:1])
			return true
		}
	}
	return false
}

func cmdAll() []cmd {
	return []cmd{
		&cmdUp{},
		&cmdDown{},
		&cmdSignal{},
	}
}

func cmdMatch(name string) cmd {
	for _, cmd := range cmdAll() {
		if cmd.Match(name) {
			return cmd
		}
	}
	return nil
}
