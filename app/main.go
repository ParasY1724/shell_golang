package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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
			} else {
				fmt.Printf("%s: not found\n", cmd)
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
		} else {
			fmt.Printf("%s: command not found\n", cmd)
		}
	}
}
