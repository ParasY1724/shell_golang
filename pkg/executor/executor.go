package executor

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/codecrafters-io/shell-starter-go/pkg/ast"
	"github.com/codecrafters-io/shell-starter-go/pkg/commands"
)

func Execute(node ast.Node, reg *commands.Registry, stdin io.Reader, stdout, stderr io.Writer) error {
	switch n := node.(type) {
	case *ast.BlockNode :
		for _,stmt := range n.Statements {
			if err := Execute(stmt,reg,stdin,stdout,stderr); err != nil {
				// break
			}
		}
		return nil
	case *ast.PipeNode:
		// Create pipe
		r, w, err := os.Pipe()
		if err != nil {
			return err
		}

		var wg sync.WaitGroup
		wg.Add(1)

		// Run Left side (write to pipe)
		go func() {
			defer wg.Done()
			defer w.Close()
			Execute(n.Left, reg, stdin, w, stderr)
		}()

		// Run Right side (read from pipe)
		err = Execute(n.Right, reg, r, stdout, stderr)
		wg.Wait()
		return err

	case *ast.RedirectNode:
		return executeRedirect(n, reg, stdin, stdout, stderr)

	case *ast.CommandNode:
		if len(n.Args) == 0 {
			return nil
		}
		return executeCommand(n.Args, reg, stdin, stdout, stderr)
	case *ast.IfNode:
		err := Execute(n.Condition, reg, stdin, stdout, stderr)
		
		if err == nil {
			return Execute(n.Then, reg, stdin, stdout, stderr)
		} else {
			if n.Else != nil {
				return Execute(n.Else, reg, stdin, stdout, stderr)
			}
		}
		return nil
	case *ast.BinaryNode:
        err := Execute(n.Left, reg, stdin, stdout, stderr)
        
        if n.Operator == "&&" {
            if err == nil {
                return Execute(n.Right, reg, stdin, stdout, stderr)
            }
            return err
        } else if n.Operator == "||" {
            if err != nil {
                return Execute(n.Right, reg, stdin, stdout, stderr)
            }
            return nil
        }
	}
	return nil
}

func executeRedirect(node *ast.RedirectNode, reg *commands.Registry, stdin io.Reader, stdout, stderr io.Writer) error {
    if node.Type == "<" { //If a user runs cat < input.txt, previous code will try to open input.txt for writing and truncate it!
        f, err := os.Open(node.Location)
        if err != nil {
            fmt.Fprintf(stderr, "error opening file: %v\n", err)
            return err
        }
        defer f.Close()
        return Execute(node.Stmt, reg, f, stdout, stderr)
    }

    // Existing logic for >, >>, 2>, etc.
    flags := os.O_CREATE | os.O_WRONLY
    if strings.HasSuffix(node.Type, ">>") {
        flags |= os.O_APPEND
    } else {
        flags |= os.O_TRUNC
    }

	f, err := os.OpenFile(node.Location, flags, 0644)
	if err != nil {
		fmt.Fprintf(stderr, "error opening file: %v\n", err)
		return err
	}
	defer f.Close()

	// Redirect specific FD
	if node.Fd == 1 {
		return Execute(node.Stmt, reg, stdin, f, stderr)
	} else if node.Fd == 2 {
		return Execute(node.Stmt, reg, stdin, stdout, f)
	}
	
	return Execute(node.Stmt, reg, stdin, stdout, stderr)
}

func executeCommand(args []string, reg *commands.Registry, stdin io.Reader, stdout, stderr io.Writer) error {
	cmdName := args[0]
	cmdArgs := args[1:]

	if fn, ok := reg.Builtins[cmdName]; ok {
		fn(cmdArgs, stdin, stdout, stderr)
		return nil
	}

	if _, err := exec.LookPath(cmdName); err == nil {
		cmd := exec.Command(cmdName, cmdArgs...)
		cmd.Stdin = stdin
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		return cmd.Run()
	}

	fmt.Fprintf(stderr, "%s: command not found\n", cmdName)
	return fmt.Errorf("not found")
}