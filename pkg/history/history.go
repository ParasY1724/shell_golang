package history

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type HistoryStruct struct {
	history      []string
	lock         sync.RWMutex
	index        int
	lastSavedIdx int 
}

func (h *HistoryStruct) LoadFile(path string) {
	if path == "" {
		return
	}
	h.lock.Lock()
	defer h.lock.Unlock()

	file, err := os.Open(path)
	if err != nil {
		return 
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			h.history = append(h.history, line)
		}
	}
	h.index = len(h.history)
}

func (h *HistoryStruct) InitFromFile(path string) {
	h.LoadFile(path)
	h.lock.Lock()
	h.lastSavedIdx = len(h.history)
	h.lock.Unlock()
}

func (h *HistoryStruct) WriteFile(path string) {
	if path == "" {
		return
	}
	h.lock.RLock()
	defer h.lock.RUnlock()

	file, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing history file: %v\n", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, cmd := range h.history {
		writer.WriteString(cmd + "\n")
	}
	writer.Flush()
	
}

func (h *HistoryStruct) AppendNew(path string) {
	if path == "" {
		return
	}
	h.lock.Lock()
	defer h.lock.Unlock()

	if h.lastSavedIdx >= len(h.history) {
		return
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error appending history file: %v\n", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for i := h.lastSavedIdx; i < len(h.history); i++ {
		writer.WriteString(h.history[i] + "\n")
	}
	writer.Flush()

	h.lastSavedIdx = len(h.history)
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