package tree

import (
	"bytes"
	"strings"
)

// ParseCursor parses a string path into a Cursor object
func ParseCursor(s string) (*Cursor, error) {
	var nodes []string
	node := bytes.NewBuffer([]byte{})
	bracketed := false

	push := func() {
		if node.Len() == 0 {
			return
		}
		s := node.String()
		if len(nodes) == 0 && s == "$" {
			node.Reset()
			return
		}

		nodes = append(nodes, s)
		node.Reset()
	}

	for pos, c := range s {
		switch c {
		case '.':
			if bracketed {
				node.WriteRune(c)
			} else {
				push()
			}

		case '[':
			if bracketed {
				return nil, SyntaxError{
					Problem:  "unexpected '['",
					Position: pos,
				}
			}
			push()
			bracketed = true

		case ']':
			if !bracketed {
				return nil, SyntaxError{
					Problem:  "unexpected ']'",
					Position: pos,
				}
			}
			push()
			bracketed = false

		default:
			node.WriteRune(c)
		}
	}
	push()

	return &Cursor{
		Nodes: nodes,
	}, nil
}

// Copy creates a copy of the cursor
func (c *Cursor) Copy() *Cursor {
	other := &Cursor{Nodes: []string{}}
	for _, node := range c.Nodes {
		other.Nodes = append(other.Nodes, node)
	}
	return other
}

// Contains checks if this cursor contains another cursor
func (c *Cursor) Contains(other *Cursor) bool {
	if len(other.Nodes) < len(c.Nodes) {
		return false
	}
	match := false
	for i := range c.Nodes {
		if c.Nodes[i] != other.Nodes[i] {
			return false
		}
		match = true
	}
	return match
}

// Under checks if this cursor is under another cursor
func (c *Cursor) Under(other *Cursor) bool {
	if len(c.Nodes) <= len(other.Nodes) {
		return false
	}
	match := false
	for i := range other.Nodes {
		if c.Nodes[i] != other.Nodes[i] {
			return false
		}
		match = true
	}
	return match
}

// Pop removes and returns the last path component
func (c *Cursor) Pop() string {
	if len(c.Nodes) == 0 {
		return ""
	}
	last := c.Nodes[len(c.Nodes)-1]
	c.Nodes = c.Nodes[0 : len(c.Nodes)-1]
	return last
}

// Push adds a path component
func (c *Cursor) Push(n string) {
	c.Nodes = append(c.Nodes, n)
}

// String returns the cursor as a dot-separated string
func (c *Cursor) String() string {
	return strings.Join(c.Nodes, ".")
}

// Depth returns the depth of the cursor path
func (c *Cursor) Depth() int {
	return len(c.Nodes)
}

// Parent returns the parent component name
func (c *Cursor) Parent() string {
	if len(c.Nodes) < 2 {
		return ""
	}
	return c.Nodes[len(c.Nodes)-2]
}

// Component returns a component by offset from the end
func (c *Cursor) Component(offset int) string {
	offset = len(c.Nodes) + offset
	if offset < 0 || offset >= len(c.Nodes) {
		return ""
	}
	return c.Nodes[offset]
}