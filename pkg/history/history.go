package history

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type HistoryStruct struct {
	history []string
	lock    sync.RWMutex
	index int
}

func (h *HistoryStruct) LoadHistory() {
	h.lock.Lock()
	defer h.lock.Unlock()

	content, err := os.ReadFile(".go_shell_history")
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error reading history file: %v\n", err)
		}
		return
	}

	rawLines := strings.Split(strings.TrimSpace(string(content)), "\n")
	
	h.history = make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		if line != "" {
			h.history = append(h.history, line)
		}
	}
	h.index = len(h.history)
}

func (h *HistoryStruct) ReadHistory(n string) {
	h.lock.RLock()
	defer h.lock.RUnlock()

	total := len(h.history)
	start := 0

	if len(n) > 0 {
		val, err := strconv.Atoi(n)
		if err != nil || val < 0 {
			fmt.Fprintf(os.Stderr, "history: numeric argument required\n")
			return
		}

		if val < total {
			start = total - val
		}
	}

	for i := start; i < total; i++ {
		fmt.Printf("\t%d  %s\n", i+1, h.history[i])
	}
}

func (h *HistoryStruct) Add(cmd string) {
	if strings.TrimSpace(cmd) == "" {
		return
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	h.history = append(h.history, cmd)
	h.index = len(h.history)
}

func (h *HistoryStruct) Save(cmd string) {
	if strings.TrimSpace(cmd) == "" {
		return
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	file, err := os.OpenFile(".go_shell_history", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Writing History File : %s\n", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(cmd + "\n"); err != nil {
		fmt.Fprintf(os.Stderr, "Error Writing History File : %s\n", err)
	}
}

func (h *HistoryStruct) GetUpEntry() (string, bool) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if h.index == 0 {
		return "", false
	}

	h.index--
	return h.history[h.index], true
}

func (h *HistoryStruct) GetDownEntry() (string, bool) {
	h.lock.Lock() 
	defer h.lock.Unlock()

	if h.index >= len(h.history) {
		return "", false 
	}

	h.index++

	if h.index == len(h.history) {
		return "", true
	}

	return h.history[h.index], true
}