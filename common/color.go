package common

import (
	"strconv"
	"strings"
)

type clrs []clr

func (c *clrs) Get(ansi ...int) clr {
	return clr{q: c, ansi: ansi}
}

func (c *clrs) Pop() clr {
	return clr{q: c, pop: true, ansi: []int{0}}
}

type clr struct {
	q    *clrs
	pop  bool
	ansi []int
}

func (c clr) str() string {
	if len(c.ansi) == 0 {
		return ""
	}
	s := make([]string, 1, 2+len(c.ansi))
	s[0] = "\033["
	for _, v := range c.ansi {
		s = append(s, strconv.Itoa(v))
	}
	s = append(s, "m")
	return strings.Join(s, "")
}

func (c clr) String() string {
	if len(c.ansi) == 0 {
		return ""
	}
	if c.pop {
		if len(*c.q) != 0 {
			*c.q = (*c.q)[:len(*c.q)-1]
			if len(*c.q) != 0 {
				l := (*c.q)[len(*c.q)-1].str()
				return l
			}
		}
		return "\033[0m"
	}

	if c.q != nil {
		*c.q = append(*c.q, c)
	}
	return c.str()
}

type stringer interface {
	String() string
}

type strStringer string

func (str strStringer) String() string {
	return string(str)
}

type stringList []stringer

func (s stringList) String() string {
	n := make([]string, len(s))
	for i := range s {
		n[i] = s[i].String()
	}
	return strings.Join(n, "")
}
