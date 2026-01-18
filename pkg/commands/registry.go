package commands

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type CmdFunc func([]string)

type Registry struct {
	Builtins map[string]CmdFunc
	ExtCmds  map[string]CmdFunc
}

func NewRegistry() *Registry {
	r := &Registry{
		Builtins: make(map[string]CmdFunc),
		ExtCmds:  make(map[string]CmdFunc),
	}
	r.registerBuiltins()
	r.registerExt()
	return r
}

func (r *Registry) registerBuiltins() {
	r.Builtins["exit"] = func(args []string) {
		os.Exit(0)
	}

	r.Builtins["echo"] = func(args []string) {
		fmt.Println(strings.Join(args, " "))
	}

	r.Builtins["type"] = func(args []string) {
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "type: missing operand")
			return
		}
		cmd := args[0]
		if _, ok := r.Builtins[cmd]; ok {
			fmt.Printf("%s is a shell builtin\n", cmd)
		} else if execPath, err := exec.LookPath(cmd); err == nil {
			fmt.Printf("%s is %s\n", cmd, execPath)
		} else {
			fmt.Fprintf(os.Stderr, "%s: not found\n", cmd)
		}
	}

	r.Builtins["pwd"] = func(args []string) {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		fmt.Println(dir)
	}

	r.Builtins["ls"] = func(args []string) {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		if dir == "-1" {
			if len(args) > 1 {
				dir = args[1]
			}
		}
		
		files, err := os.ReadDir(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ls: %s: No such file or directory\n", dir)
			return
		}

		for _, file := range files {
			fmt.Println(file.Name())
		}
	}

	r.Builtins["cd"] = func(args []string) {
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "cd: missing argument")
			return
		}

		dir := args[0]
		if dir[0] == '~' {
			if homeDir, er := os.UserHomeDir(); er == nil {
				dir = strings.Replace(dir, "~", homeDir, 1)
			} else {
				fmt.Println(er)
			}
		}

		info, err := os.Stat(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", dir)
			return
		}

		if !info.IsDir() {
			fmt.Fprintf(os.Stderr, "cd: %s: Not a directory\n", dir)
			return
		}

		if err := os.Chdir(dir); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func (r *Registry) registerExt() {
	r.ExtCmds["cat"] = func(args []string) {
		for _, filename := range args {
			f, err := os.Open(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "cat: %s: No such file or directory\n", filename)
				continue
			}

			_, err = io.Copy(os.Stdout, f)
			f.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "cat: %s: read error\n", filename)
			}
		}
	}
}