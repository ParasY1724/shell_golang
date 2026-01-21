package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/codecrafters-io/shell-starter-go/pkg/commands"
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

		//  Prepare for Pipeline Execution
		var prevPipeReader *os.File = nil
		var wg sync.WaitGroup

		for i, parts := range pipelineCmds {
			if len(parts) == 0 {
				continue
			}

			var outFile, errFile string
			redirectOut, redirectErr, appendOut, appendErr := false, false, false, false
			
			args := parts
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

			execStdin := os.Stdin
			if prevPipeReader != nil {
				execStdin = prevPipeReader
			}

			execStdout := os.Stdout
			if nextPipeWriter != nil {
				execStdout = nextPipeWriter
			}
			
			execStderr := os.Stderr

			if redirectOut {
				flags := os.O_CREATE | os.O_WRONLY
				if appendOut {
					flags |= os.O_APPEND
				} else {
					flags |= os.O_TRUNC
				}
				f, err := os.OpenFile(outFile, flags, 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
				} else {
					execStdout = f
				}
			}

			if redirectErr {
				flags := os.O_CREATE | os.O_WRONLY
				if appendErr {
					flags |= os.O_APPEND
				} else {
					flags |= os.O_TRUNC
				}
				f, err := os.OpenFile(errFile, flags, 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
				} else {
					execStderr = f
				}
			}

			closeResources := func(in, out, errFile *os.File) {
				if in != os.Stdin && in != nil { in.Close() }
				if out != os.Stdout && out != nil { out.Close() }
				if errFile != os.Stderr && errFile != nil { errFile.Close() }
			}

			if fn, ok := registry.Builtins[cmdName]; ok {
				wg.Add(1)
				go func(in, out, errFile *os.File, args []string, fn commands.CmdFunc) {
					defer wg.Done()
					
					defer closeResources(in, out, errFile)

					builtinLock.Lock()
					defer builtinLock.Unlock()

					oldStdout := os.Stdout
					oldStdin := os.Stdin
					oldStderr := os.Stderr

					defer func() {
						os.Stdout = oldStdout
						os.Stdin = oldStdin
						os.Stderr = oldStderr
					}()

					if in != nil { os.Stdin = in }
					if out != nil { os.Stdout = out }
					if errFile != nil { os.Stderr = errFile }

					fn(args)
				}(execStdin, execStdout, execStderr, cmdArgs, fn)
				

			} else {
				cmd := exec.Command(cmdName, cmdArgs...)
				cmd.Stdin = execStdin
				cmd.Stdout = execStdout
				cmd.Stderr = execStderr

				if err := cmd.Start(); err != nil {
					fmt.Printf("%s: command not found\n", cmdName)
					closeResources(execStdin, execStdout, execStderr)
				} else {
					closeResources(execStdin, execStdout, execStderr)

					wg.Add(1)
					go func() {
						defer wg.Done()
						cmd.Wait()
					}()
				}
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