package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/codecrafters-io/shell-starter-go/pkg/commands"
	"github.com/codecrafters-io/shell-starter-go/pkg/executor"
	"github.com/codecrafters-io/shell-starter-go/pkg/lexer"
	"github.com/codecrafters-io/shell-starter-go/pkg/parser"
	"github.com/codecrafters-io/shell-starter-go/pkg/term"
	"github.com/codecrafters-io/shell-starter-go/pkg/utils"
)

func main() {
	registry := commands.NewRegistry()

	histFile := os.Getenv("HISTFILE")
	if histFile != "" {
		registry.History.InitFromFile(histFile, os.Stderr)
	}

	oldState, err := term.EnableRawMode(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.RestoreTerminal(int(os.Stdin.Fd()), oldState)

	sigChan := make(chan os.Signal, 1) // SIGINT CTRL + C
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		term.RestoreTerminal(int(os.Stdin.Fd()), oldState)
		if histFile != "" {
			registry.History.WriteFile(histFile, os.Stderr)
		}
		fmt.Print("\n")
		os.Exit(0)
	}()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("$ ")

		var line strings.Builder
		tabCount := 0

		// --- INPUT LOOP (Preserved Exact Behaviour) ---
		for {
			ch, err := reader.ReadByte()
			if err != nil {
				return
			}

			switch ch {
			case '\n', '\r': // ENTER
				fmt.Println()
				goto EXECUTE

			case '\t': // TAB (Autocomplete using Trie)
				input := line.String()
				parts := strings.Split(input, " ")
				lastWord := parts[len(parts)-1]

				if len(lastWord) > 0 {
					suggestion, found := registry.Suggest(lastWord)
					if found && len(suggestion) > 0 {
						lcp := utils.FindLeastPrefix(suggestion)
						if len(lcp) > len(lastWord) {
							suffix := lcp[len(lastWord):]
							line.WriteString(suffix)
							fmt.Print(suffix)
							tabCount = 0
						}
						if len(suggestion) == 1 {
							line.WriteString(" ")
							fmt.Print(" ")
							tabCount = 0
						} else {
							if len(lcp) == len(lastWord) {
								tabCount++
								if tabCount == 1 {
									fmt.Print("\x07")
								} else {
									fmt.Print("\r\n")
									fmt.Println(strings.Join(suggestion, "  "))
									fmt.Print("$ ", line.String())
									tabCount = 0
								}
							}
						}
					} else {
						fmt.Print("\x07")
					}
				}
			case 127: // BACKSPACE
				if line.Len() > 0 {
					s := line.String()
					line.Reset()
					line.WriteString(s[:len(s)-1])
					fmt.Print("\b \b")
				}

			case 27: // Esc (Arrows)
				if b1, err := reader.ReadByte(); err == nil && b1 == '[' {
					if b2, err := reader.ReadByte(); err == nil {
						var histCmd string
						var ok bool

						switch b2 {
						case 'A': // UP ARROW
							histCmd, ok = registry.History.GetUpEntry()
						case 'B': // DOWN ARROW
							histCmd, ok = registry.History.GetDownEntry()
						}

						if !ok {
							fmt.Print("\x07")
						} else {
							fmt.Print("\033[2K\r$ ")
							line.Reset()
							line.WriteString(histCmd)
							fmt.Print(histCmd)
						}
					}
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

		registry.History.Add(cmdLine)

		// --- NEW ARCHITECTURE ---
		// 1. Lexing
		l := lexer.New(cmdLine)
		
		// 2. Parsing (Build AST)
		p := parser.New(l)
		program := p.Parse()

		// 3. Execution (Recursively Walk AST)
		if program != nil {
			err := executor.Execute(program, registry, os.Stdin, os.Stdout, os.Stderr)
			if err != nil {
				// Errors are printed inside executor or builtins usually
			}
		}

		if registry.ExitSignal {
			if histFile != "" {
				registry.History.WriteFile(histFile, os.Stderr)
			}
			return
		}
	}
}