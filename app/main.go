package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"os/exec"
	"io"
)

var buildinCmds map[string]func([]string)
var extCmds map[string]func([]string)


func init() {
	buildinCmds = map[string]func([]string){
		"exit": func(args []string) {
			os.Exit(0)
		},
		"echo": func(args []string) {
			fmt.Println(strings.Join(args, " "))
		},
		"type": func(args []string) {
			if len(args) == 0 {
				fmt.Println("type: missing operand")
				return
			}
			cmd := args[0]
			if _, ok := buildinCmds[cmd]; ok {
				fmt.Printf("%s is a shell builtin\n", cmd)
				
			} else if execPath, err := exec.LookPath(cmd); err == nil {
				fmt.Printf("%s is %s\n",cmd,execPath)
			} else {
				fmt.Printf("%s: not found\n",cmd)
			}		
		},
		"pwd" : func(args []string) {
			dir , err := os.Getwd()
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(dir)
		},
		"cd" : func(args []string) {
			if len(args) == 0 {
				fmt.Fprintln(os.Stderr, "cd: missing argument")
				return
			}

			dir := args[0]
			if dir[0] == '~' {
				if homeDir,er := os.UserHomeDir();er == nil {
					dir = strings.Replace(dir,"~",homeDir,1)
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
		},
	}
	

	extCmds = map[string]func([]string) {
		"cat" : func(args []string) {
			for _,filename := range args {
				f,err := os.Open(filename)
				if err != nil {
					fmt.Fprintf(os.Stderr, "cat: %s\n", err)
					break
				}
				if _, err := io.Copy(os.Stdout, f); err != nil {
					fmt.Fprintf(os.Stderr, "cat: error reading: %s\n", err)
					break
				}
				f.Close()
			}
		},
	} 
}


func parseInput(line string) []string {
	var args []string
	var current strings.Builder

	inSingle := false
	inDouble := false
	escaped := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			if inSingle {
				current.WriteByte(ch)
			} else if inDouble {
				if i+1 < len(line) {
					next := line[i+1]
					if next == '"' || next == '\\' || next == '$' || next == '`' || next == '\n' {
						escaped = true
					} else {
						current.WriteByte(ch)
					}
				}
			} else {
				escaped = true
			}
			continue
		}

		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}

		if (ch == ' ' || ch == '\t') && !inSingle && !inDouble {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(ch)
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}



func main() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("$ ")

		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		
		if line == "" {
			continue
		}

		parts := parseInput(line)
		cmd := parts[0]
		args := parts[1:]

		if fn, ok := buildinCmds[cmd]; ok {
			fn(args)
		} else if fn,ok = extCmds[cmd]; ok {
			fn(args)
		}else if _, err := exec.LookPath(cmd); err == nil {
			externalCmd := exec.Command(cmd, args...)
			externalCmd.Stdout = os.Stdout
			externalCmd.Stderr = os.Stderr
			externalCmd.Stdin = os.Stdin
			if err := externalCmd.Run(); err != nil {
				fmt.Printf("%s: error running command: %v\n", cmd, err)
			}
		} else {
			fmt.Printf("%s: command not found\n", cmd)
		}
	}
}
