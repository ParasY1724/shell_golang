package commands

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type CmdFunc func([]string)

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
	Builtins map[string]CmdFunc
	CmdTrie  *Trie 
}

func NewRegistry() *Registry {
	r := &Registry{
		Builtins: make(map[string]CmdFunc),
		CmdTrie:  NewTrie(),
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
		return []string{""} , false
	}
	
	sort.Strings(candidates)

	return candidates, true
}

func (r *Registry) registerBuiltins() {
	add := func(name string, fn CmdFunc) {
		r.Builtins[name] = fn
		r.CmdTrie.Insert(name)
	}

	add("exit", func(args []string) {
		os.Exit(0)
	})

	add("echo", func(args []string) {
		fmt.Println(strings.Join(args, " "))
	})

	add("type", func(args []string) {
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
	})

	add("pwd", func(args []string) {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		fmt.Println(dir)
	})

	add("ls", func(args []string) {
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
	})

	add("cd", func(args []string) {
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
	})

	add("history", func(args []string) {
		content, err := os.ReadFile(".go_shell_history")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading history file: %v\n", err)
			return
		}
	
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		total := len(lines)
	
		start := 0
	
		if len(args) > 0 {
			n, err := strconv.Atoi(args[0])
			if err != nil || n < 0 {
				fmt.Fprintf(os.Stderr, "Error: expected positive integer\n")
				return
			}
	
			if n < total {
				start = total - n
			}
		}
	
		for i := start; i < total; i++ {
			fmt.Printf("\t%d  %s\n", i+1, lines[i])
		}
	})
	
}
