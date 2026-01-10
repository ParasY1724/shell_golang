package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var knownCmds = map[string]func([]string){
	"exit": func(args []string) {
		os.Exit(0)
	},
	"echo": func(args []string) {
		fmt.Println(strings.Join(args, " "))
	},
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
