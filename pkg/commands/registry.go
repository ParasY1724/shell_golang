package commands

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/codecrafters-io/shell-starter-go/pkg/history"
)

type CmdFunc func(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer)
type TrieNode struct {
	children map[rune]*TrieNode
	isEnd    bool
	word     string
}

type Trie struct {
	root *TrieNode
}

func NewTrie() *Trie {
	return &Trie{root: &TrieNode{children: make(map[rune]*TrieNode)}}
}

func (t *Trie) Insert(word string) {
	node := t.root
	for _, ch := range word {
		if _, ok := node.children[ch]; !ok {
			node.children[ch] = &TrieNode{children: make(map[rune]*TrieNode)}
		}
		node = node.children[ch]
	}
	node.isEnd = true
	node.word = word
}

func (t *Trie) SearchPrefix(prefix string) []string {
	node := t.root
	for _, ch := range prefix {
		if _, ok := node.children[ch]; !ok {
			return nil
		}
		node = node.children[ch]
	}
	return t.collect(node)
}

func (t *Trie) collect(node *TrieNode) []string {
	var results []string
	if node.isEnd {
		results = append(results, node.word)
	}
	for _, child := range node.children {
		results = append(results, t.collect(child)...)
	}
	return results
}

type Registry struct {
	Builtins   map[string]CmdFunc
	CmdTrie    *Trie
	History    *history.HistoryStruct
	ExitSignal bool
}

func NewRegistry() *Registry {
	r := &Registry{
		Builtins: make(map[string]CmdFunc),
		CmdTrie:  NewTrie(),
		History:  &history.HistoryStruct{},
	}
	r.registerBuiltins()
	r.loadPathExecutables()

	return r
}

func (r *Registry) loadPathExecutables() {
	pathEnv := os.Getenv("PATH")
	paths := strings.Split(pathEnv, string(os.PathListSeparator))

	for _, dir := range paths {
		if dir == "" {
			continue
		}

		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			info, err := file.Info()
			if err != nil {
				continue
			}
			if info.Mode()&0111 != 0 {
				r.CmdTrie.Insert(file.Name())
			}
		}
	}
}

func (r *Registry) Suggest(prefix string) ([]string, bool) {
	candidates := r.CmdTrie.SearchPrefix(prefix)

	if len(candidates) == 0 {
		return []string{""}, false
	}

	sort.Strings(candidates)

	return candidates, true
}

func (r *Registry) registerBuiltins() {
	add := func(name string, fn CmdFunc) {
		r.Builtins[name] = fn
		r.CmdTrie.Insert(name)
	}

	add("exit", func(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) {
			r.ExitSignal = true
	})

	add("echo", func(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) {
		fmt.Println(stdout,strings.Join(args, " "))
	})

	add("type", func(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) {
		if len(args) == 0 {
			fmt.Fprintln(stderr, "type: missing operand")
			return
		}
		cmd := args[0]
		if _, ok := r.Builtins[cmd]; ok {
			fmt.Fprintf(stdout, "%s is a shell builtin\n", cmd)
		} else if execPath, err := exec.LookPath(cmd); err == nil {
			fmt.Fprintf(stdout, "%s is %s\n", cmd, execPath)
		} else {
			fmt.Fprintf(stderr, "%s: not found\n", cmd)
		}
	})

	add("pwd", func(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return
		}
		fmt.Fprintln(stdout, dir)
	})

	add("ls", func(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		if dir == "-1" {
			if len(args) > 1 {
				dir = args[1]
			} else {
				dir = "."
			}
		}

		files, err := os.ReadDir(dir)
		if err != nil {
			fmt.Fprintf(stderr, "ls: %s: No such file or directory\n", dir)
			return
		}

		for _, file := range files {
			fmt.Fprintln(stdout, file.Name()) // Writes to pipe if connected
		}
	})

	add("cd", func(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) {
		if len(args) == 0 {
			fmt.Fprintln(stderr, "cd: missing argument")
			return
		}

		dir := args[0]
		if dir[0] == '~' {
			if homeDir, er := os.UserHomeDir(); er == nil {
				dir = strings.Replace(dir, "~", homeDir, 1)
			} else {
				fmt.Fprintln(stdout, er)
			}
		}

		info, err := os.Stat(dir)
		if err != nil {
			fmt.Fprintf(stderr, "cd: %s: No such file or directory\n", dir)
			return
		}

		if !info.IsDir() {
			fmt.Fprintf(stderr, "cd: %s: Not a directory\n", dir)
			return
		}

		if err := os.Chdir(dir); err != nil {
			fmt.Fprintln(stderr, err)
		}
	})

	add("history", func(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) {
		if len(args) > 0 {

			arg := args[0]
			if len(args) >= 2 && (arg == "-r" || arg == "-w" || arg == "-a") {
				path := args[1]
				switch arg {
				case "-r":
					r.History.LoadFile(path,stderr)
				case "-w":
					r.History.WriteFile(path,stderr)
				case "-a":
					r.History.AppendNew(path,stderr)
				}
				return
			}
			r.History.ReadHistory(arg,stdout,stderr)
		} else {
			r.History.ReadHistory("",stdout,stderr)
		}
	})

}