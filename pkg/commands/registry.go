package commands

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sort"
	"syscall"
	"github.com/codecrafters-io/shell-starter-go/pkg/history"
	"github.com/codecrafters-io/shell-starter-go/pkg/utils"
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

type Job struct {
	ID      int
	PID     int
	Command string
	Cmd     *exec.Cmd // nil for complex/pipeline jobs
}



type Registry struct {
	Builtins   map[string]CmdFunc
	CmdTrie    *Trie
	History    *history.HistoryStruct
	ExitSignal bool

	Jobs      map[int]*Job
	JobMutex  sync.Mutex
}

func NewRegistry() *Registry {
	r := &Registry{
		Builtins: make(map[string]CmdFunc),
		CmdTrie:  NewTrie(),
		History:  &history.HistoryStruct{},
		Jobs:     make(map[int]*Job),
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
				r.CmdTrie.Insert(file.Name() + "/")
				continue
			}
			info, err := file.Info()
			if err != nil {
				continue
			}
			if info.Mode()&0111 != 0 {
				r.CmdTrie.Insert(file.Name() + " ")
			}
		}
	}
}

func (r *Registry) Suggest(prefix string) ([]string, bool) {
	candidates := r.CmdTrie.SearchPrefix(prefix)

	if len(candidates) == 0 {
		return []string{}, false
	}

	return candidates, true
}

func (r *Registry) registerBuiltins() {
	add := func(name string, fn CmdFunc) {
		r.Builtins[name] = fn
		r.CmdTrie.Insert(name + " ")
	}

	add("exit", func(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) {
		r.ExitSignal = true
	})

	add("echo", func(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) {
		fmt.Fprintln(stdout, strings.Join(args, " "))
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
					r.History.LoadFile(path, stderr)
				case "-w":
					r.History.WriteFile(path, stderr)
				case "-a":
					r.History.AppendNew(path, stderr)
				}
				return
			}
			r.History.ReadHistory(arg, stdout, stderr)
		} else {
			r.History.ReadHistory("", stdout, stderr)
		}
	})

	add("jobs" , func(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer){
		r.ReapJobs(stdout)

		r.JobMutex.Lock()
		defer r.JobMutex.Unlock()

		var ids []int
		for id := range r.Jobs {
			ids = append(ids, id)
		}
		sort.Ints(ids)

		for i, id := range ids {
			job := r.Jobs[id]
			sign := utils.MarkerForIndex(i, len(ids))
			fmt.Fprintf(stdout, "[%d]%s  Running                 %s &\n", job.ID, sign, job.Command)
		}
	})

}

func (r *Registry) SuggestFilename(token string) ([]string, bool) {
	var searchDir, prefix string
	var isLocal bool

	lastSlash := strings.LastIndex(token, "/")
	if lastSlash != -1 {
		searchDir = token[:lastSlash+1] // Includes the trailing '/' (e.g., "path/to/")
		prefix = token[lastSlash+1:]    // Everything after the '/'
	} else {
		searchDir = "."
		prefix = token
		isLocal = true
	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return nil, false
	}

	var candidates []string

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {

			var match string
			if isLocal {
				match = entry.Name()
			} else {
				match = searchDir + entry.Name()
			}

			if entry.IsDir() {
				candidates = append(candidates, match+"/")
			} else {
				candidates = append(candidates, match+" ")
			}
		}
	}

	if len(candidates) == 0 {
		return nil, false
	}

	return candidates, true
}



func (r *Registry) AddJob(pid int, cmd string, execCmd *exec.Cmd) int {
	r.JobMutex.Lock()
	defer r.JobMutex.Unlock()
	
	id := 1
	for {
		if _, exists := r.Jobs[id]; !exists {
			break
		}
		id++
	}

	r.Jobs[id] = &Job{ID: id, PID: pid, Command: cmd, Cmd: execCmd}
	return id
}

func (r *Registry) RemoveJob(id int) {
	r.JobMutex.Lock()
	defer r.JobMutex.Unlock()
	delete(r.Jobs, id)
}

// ReapJobs non-blocking checks all background jobs.
// Done jobs are printed then removed; Running jobs are left for the next `jobs` call.
func (r *Registry) ReapJobs(stdout io.Writer) {
	r.JobMutex.Lock()
	defer r.JobMutex.Unlock()

	var ids []int
	for id := range r.Jobs {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	total := len(ids)

	//determine which jobs are done
	doneSet := make(map[int]bool)
	for _, id := range ids {
		job := r.Jobs[id]
		if job.Cmd != nil && job.PID > 0 {
			var ws syscall.WaitStatus
			wpid, err := syscall.Wait4(job.PID, &ws, syscall.WNOHANG, nil)
			if err != nil || (wpid == job.PID && (ws.Exited() || ws.Signaled())) {
				doneSet[id] = true
			}
		}
	}

	// Second pass: print Done jobs with markers based on position in full list, then remove
	for i, id := range ids {
		if doneSet[id] {
			sign := utils.MarkerForIndex(i, total)
			fmt.Fprintf(stdout, "[%d]%s  Done                    %s\n", r.Jobs[id].ID, sign, r.Jobs[id].Command)
			delete(r.Jobs, id)
		}
	}
}