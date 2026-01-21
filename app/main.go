package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/codecrafters-io/shell-starter-go/pkg/commands"
	"github.com/codecrafters-io/shell-starter-go/pkg/executor"
	"github.com/codecrafters-io/shell-starter-go/pkg/parser"
	"github.com/codecrafters-io/shell-starter-go/pkg/term"
	"github.com/codecrafters-io/shell-starter-go/pkg/utils"
)

var builtinLock sync.Mutex

func main() {
	registry := commands.NewRegistry()

	histFile := os.Getenv("HISTFILE")
	if histFile != "" {
		registry.History.InitFromFile(histFile)
	}

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

			case 27: // Esc
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
							fmt.Print("\033[2K\r$ ") // \033[2K clears line, \r moves cursor to start
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

		registry.History.Add(cmdLine)

		allParts := parser.ParseInput(cmdLine)
		if len(allParts) == 0 {
			continue
		}

		var pipelineCmds [][]string
		var currentCmd []string
		for _, token := range allParts {
			if token == "|" {
				if len(currentCmd) > 0 {
					pipelineCmds = append(pipelineCmds, currentCmd)
				}
				currentCmd = nil
			} else {
				currentCmd = append(currentCmd, token)
			}
		}
		if len(currentCmd) > 0 {
			pipelineCmds = append(pipelineCmds, currentCmd)
		}

		var prevPipeReader *os.File = nil
		var wg sync.WaitGroup

		for i, parts := range pipelineCmds {
			if len(parts) == 0 {
				continue
			}

			redirectOut := false
			redirectErr := false
			appendOut := false
			appendErr := false
			var outFile, errFile string

			args := parts
			// Handle Redirection parsing
			if len(args) >= 3 {
				op := args[len(args)-2]
				file := args[len(args)-1]
				handled := true
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
				default:
					handled = false
				}
				if handled {
					args = args[:len(args)-2]
				}
			}

			cmdName := args[0]
			cmdArgs := args[1:]

			var nextPipeReader *os.File
			var nextPipeWriter *os.File
			var err error

			if i < len(pipelineCmds)-1 {
				nextPipeReader, nextPipeWriter, err = os.Pipe()
				if err != nil {
					fmt.Fprintf(os.Stderr, "pipe error: %v\n", err)
					break
				}
			}

			var effectiveStdin *os.File = os.Stdin
			if prevPipeReader != nil {
				effectiveStdin = prevPipeReader
			}

			var effectiveStdout *os.File = os.Stdout
			if nextPipeWriter != nil {
				effectiveStdout = nextPipeWriter
			}

			thisCmdName := cmdName
			thisArgs := cmdArgs
			thisStdin := effectiveStdin
			thisStdout := effectiveStdout
			
			// Flags for closure
			isRedirectOut := redirectOut
			isRedirectErr := redirectErr
			fOut := outFile
			fErr := errFile
			isAppOut := appendOut
			isAppErr := appendErr

			closeStdin := (prevPipeReader != nil)
			closeStdout := (nextPipeWriter != nil)

			wg.Add(1)
			go func() {
				defer wg.Done()

				if closeStdin && thisStdin != nil {
					defer thisStdin.Close()
				}
				if closeStdout && thisStdout != nil {
					defer thisStdout.Close()
				}

				run := func() {
					if fn, ok := registry.Builtins[thisCmdName]; ok {
						builtinLock.Lock()
						
						defer builtinLock.Unlock()

						oldStdout := os.Stdout
						oldStdin := os.Stdin

						defer func() {
							os.Stdout = oldStdout
							os.Stdin = oldStdin
						}()

						os.Stdout = thisStdout
						os.Stdin = thisStdin

						fn(thisArgs)
					} else if _, err := exec.LookPath(thisCmdName); err == nil {
						c := exec.Command(thisCmdName, thisArgs...)
						c.Stdin = thisStdin
						c.Stdout = thisStdout
						c.Stderr = os.Stderr
						c.Run()
					} else {
						fmt.Printf("%s: command not found\n", thisCmdName)
					}
				}

				if isRedirectOut && isRedirectErr {
					executor.WithStdRedirect(fOut, isAppOut, false, func() {
						executor.WithStdRedirect(fErr, isAppErr, true, run)
					})
				} else if isRedirectOut {
					executor.WithStdRedirect(fOut, isAppOut, false, run)
				} else if isRedirectErr {
					executor.WithStdRedirect(fErr, isAppErr, true, run)
				} else {
					run()
				}
			}()

			// PARENT THREAD MANAGEMENT
			// Crucial: The parent must close its copies of the pipes.
			// We are done with prevPipeReader (passed to child).
			if prevPipeReader != nil {
				prevPipeReader.Close()
			}
			// We are done with nextPipeWriter (passed to child).
			// If we don't close this, the next command (reader) will never get EOF.
			if nextPipeWriter != nil {
				nextPipeWriter.Close()
			}

			prevPipeReader = nextPipeReader
		}

		wg.Wait()

		if registry.ExitSignal {
			if histFile != "" {
				registry.History.WriteFile(histFile)
			}
			return
		}
	}
}