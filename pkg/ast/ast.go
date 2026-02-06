package ast

import "strings"

type Node interface {
	String() string
}

type CommandNode struct {
	Args []string
}

type PipeNode struct {
	Left  Node
	Right Node
}

type RedirectNode struct {
	Stmt     Node
	Location string // Filename
	Type     string // >, >>, 1>, 2>
	Fd       int    // 1 for stdout, 2 for stderr
}

func (c *CommandNode) String() string {
	return strings.Join(c.Args, " ")
}

func (p *PipeNode) String() string {
	return " | "
}

func (r *RedirectNode) String() string {
	return " " + r.Type + " " + r.Location
}