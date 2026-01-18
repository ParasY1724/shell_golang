package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/codecrafters-io/shell-starter-go/pkg/commands"
	"github.com/codecrafters-io/shell-starter-go/pkg/executor"
	"github.com/codecrafters-io/shell-starter-go/pkg/parser"
	"github.com/codecrafters-io/shell-starter-go/pkg/term"
)

func main() {
	registry := commands.NewRegistry()

	oldState, err := term.EnableRawMode(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.RestoreTerminal(int(os.Stdin.Fd()), oldState)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("$ ")

		var line strings.Builder
		tabCount := 0

		for {
			ch, err := reader.ReadByte()
			if err != nil {
				return
			}

			switch ch {
			case '\n','\r': // ENTER
				fmt.Println()
				goto EXECUTE

				case '\t': // TAB (Autocomplete using Trie)
					input := line.String()
					
					parts := strings.Split(input, " ")
					lastWord := parts[len(parts)-1]
					
					if len(lastWord) > 0 {
						suggestion, found := registry.Suggest(lastWord)
						if found {
							
							if (len(suggestion) == 1){
								suffix := suggestion[0][len(lastWord):]
								line.WriteString(suffix + " ")
								fmt.Print(suffix + " ")
								tabCount = 0
							} else {
								tabCount++;
								if (tabCount == 1){
									fmt.Print("\x07")
								} else {
									fmt.Print("\r\n")
									fmt.Println(strings.Join(suggestion, "  "))
									fmt.Print("$ ", line.String())
									tabCount = 0
								}
							}
							
						} else {
							fmt.Print("\x07")
						}
					}

			case 127: // BACKSPACE '/'
				if line.Len() > 0 {
					s := line.String()
					line.Reset()
					line.WriteString(s[:len(s)-1])
					fmt.Print("\b \b")
				}

			default:
				line.WriteByte(ch)
				fmt.Print(string(ch))
			}
		}

	EXECUTE:
		cmdLine := strings.TrimSpace(line.String())
		if cmdLine == "" {
			continue
		}

		parts := parser.ParseInput(cmdLine)
		if len(parts) == 0 {
			continue
		}
		
		cmd := parts[0]
		args := parts[1:]

		// Handle Redirection logic
		redirectOut := false
		redirectErr := false
		appendOut := false
		appendErr := false
		var outFile, errFile string

		if len(args) >= 2 {
			op := args[len(args)-2]
			file := args[len(args)-1]

			switch op {
			case ">", "1>":
				redirectOut = true
				outFile = file
			case ">>", "1>>":
				redirectOut = true
				appendOut = true
				outFile = file
			case "2>":
				redirectErr = true
				errFile = file
			case "2>>":
				redirectErr = true
				appendErr = true
				errFile = file
			}

			if redirectOut || redirectErr {
				args = args[:len(args)-2]
			}
		}

		run := func() {
			if fn, ok := registry.Builtins[cmd]; ok {
				fn(args)
			} else if fn, ok := registry.ExtCmds[cmd]; ok {
				fn(args)
			} else if _, err := exec.LookPath(cmd); err == nil {
				// System command fallback
				c := exec.Command(cmd, args...)
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				c.Stdin = os.Stdin
				c.Run()
			} else {
				fmt.Printf("%s: command not found\n", cmd)
			}
		}

		if redirectOut && redirectErr {
			executor.WithStdRedirect(outFile, appendOut, false, func() {
				executor.WithStdRedirect(errFile, appendErr, true, run)
			})
		} else if redirectOut {
			executor.WithStdRedirect(outFile, appendOut, false, run)
		} else if redirectErr {
			executor.WithStdRedirect(errFile, appendErr, true, run)
		} else {
			run()
		}
	}
}