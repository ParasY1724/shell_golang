package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"os/exec"
)

var knownCmds map[string]func([]string)

func init() {
	knownCmds = map[string]func([]string){
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
			if _, ok := knownCmds[cmd]; ok {
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
			if _,err := os.Stat(args[0]) ; os.IsNotExist(err){
				fmt.Printf("cd: %s: No such file or directory\n",args[0])
				return
			}
			err := os.Chdir(args[0])
			if err != nil {
				println(err)
			}
		},
	}
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

		parts := strings.Fields(line)
		cmd := parts[0]
		args := parts[1:]

		if fn, ok := knownCmds[cmd]; ok {
			fn(args)
		} else if _, err := exec.LookPath(cmd); err == nil {
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
