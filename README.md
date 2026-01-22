[![progress-banner](https://backend.codecrafters.io/progress/shell/d51cb964-13cf-4f25-84ab-e0337ede75ab)](https://app.codecrafters.io/users/codecrafters-bot?r=2qF)


CodeCrafter's ["Build Your Own Shell" Challenge](https://app.codecrafters.io/courses/shell/overview).


---

# Go Shell Implementation

A custom, POSIX-compliant shell implementation written in Go. This project demonstrates low-level terminal control, command parsing, pipeline management, and built-in command handling.

## 📂 Project Structure

The project is organized into a modular structure separating the REPL logic, command parsing, execution, and terminal management.

```text
.
├── main.go                # Entry point: REPL loop, pipeline orchestration, and raw mode setup
└── pkg/
    ├── commands/
    │   └── registry.go    # Command registry, built-in definitions, and Trie-based autocomplete
    ├── history/
    │   └── history.go     # Command history management (in-memory & file persistence)
    ├── parser/
    │   └── parser.go      # Input tokenizer handling quotes (' " \), spaces, and pipes
    ├── term/
    │   └── term.go        # Low-level terminal control (Raw mode vs Canonical mode)
    └── executor/
        └── redirect.go    # Utilities for handling standard I/O redirection

```

---

## 🧩 Component Breakdown

### 1. Core Logic (`main.go`)

The heart of the shell. It manages the lifecycle of the application:

* **REPL Loop:** Reads input byte-by-byte to handle special keys (Tab, Arrow keys) in real-time.
* **Pipeline Orchestration:** Parses commands separated by `|`, creates `os.Pipe` connections between them, and manages `sync.WaitGroup` for concurrent execution.
* **Signal Handling:** Detects special keys like `Ctrl+C` or `<Arrow Keys>` for history navigation.

### 2. Command Registry (`pkg/commands`)

* **`registry.go`**: Maintains a map of built-in commands (`cd`, `exit`, `type`, `echo`, `pwd`, `history`).
* **Trie Implementation:** Uses a prefix tree (Trie) to index all executables in the system `$PATH` and built-ins. This powers the **Tab Autocomplete** feature.

### 3. Input Parser (`pkg/parser`)

* **`parser.go`**: specialized tokenizer that converts raw input strings into executable tokens.
* **Quote Handling**: Correctly handles single quotes `'`, double quotes `"`, and backslash escapes `\`, ensuring arguments with spaces are grouped correctly (e.g., `"echo 'hello world'"` becomes `["echo", "hello world"]`).

### 4. Terminal Control (`pkg/term`)

* **`term.go`**: Uses `syscall` to switch the terminal from **Canonical Mode** (buffered input) to **Raw Mode**.
* **Why?** This is required to capture keystrokes like `Tab` (for autocomplete) and `Up/Down Arrows` (for history) immediately, without waiting for the user to press Enter.

### 5. History Management (`pkg/history`)

* **`history.go`**: Manages a thread-safe list of previous commands.
* **Persistence**: Reads from and writes to the file defined in `HISTFILE` (similar to `.bash_history`).
* **Navigation**: Provides methods `GetUpEntry()` and `GetDownEntry()` for scrolling through commands.

### 6. Execution & Redirection

* **Redirect Logic (in `main.go` & `pkg/executor`)**: Handles standard file descriptors manipulation for:
* `>` (Overwrite stdout)
* `>>` (Append stdout)
* `2>` (Overwrite stderr)
* `2>>` (Append stderr)



---

## 🚀 Features

* **Pipeline Support:** Run commands in parallel like `ls | grep .go | sort`.
* **I/O Redirection:** Support for `stdout` and `stderr` redirection (e.g., `ls > file.txt 2> error.log`).
* **Tab Autocomplete:** Intelligently suggests commands based on `$PATH` and built-ins.
* **Command History:** Persistent history with Arrow key navigation.
* **Built-ins:**
   * `cd`: Change directory (supports `~`).
   * `type`: Reveal if a command is a built-in or executable path.
   * `history`: View or manipulate session history.
   * `echo`: Print arguments to stdout.


## 🛠 Usage

To build and run the shell:

```bash
# Build the binary
go build -o myshell main.go

# Run the shell
./myshell
```
