[![progress-banner](https://backend.codecrafters.io/progress/shell/d51cb964-13cf-4f25-84ab-e0337ede75ab)](https://app.codecrafters.io/users/codecrafters-bot?r=2qF)

---

# Go Shell Implementation

A custom, POSIX-compliant shell implementation written in Go. This project demonstrates low-level terminal control, AST-based command parsing, pipeline management, control flow, and built-in command handling.

## 📂 Project Structure

```text
.
├── main.go                # Entry point: REPL loop, pipeline orchestration, and raw mode setup
└── pkg/
    ├── ast/
    │   └── ast.go         # AST node definitions: Command, Pipe, Redirect, If, Block, Binary
    ├── lexer/
    │   └── lexer.go       # Tokenizer: handles quotes, escapes, redirects, &&, ||, pipes
    ├── parser/
    │   └── parser.go      # Recursive-descent parser that builds the AST from tokens
    ├── token/
    │   └── token.go       # Token type definitions and keyword lookup table
    ├── commands/
    │   └── registry.go    # Command registry, built-in definitions, and Trie-based autocomplete
    ├── history/
    │   └── history.go     # Command history management (in-memory & file persistence)
    ├── term/
    │   └── term.go        # Low-level terminal control (Raw mode vs Canonical mode)
    └── executor/
        └── executor.go    # AST walker: executes nodes recursively with I/O redirection
```

---

## 🧩 Component Breakdown

### 1. Core Logic (`main.go`)

The heart of the shell. It manages the lifecycle of the application:

- **REPL Loop:** Reads input byte-by-byte to handle special keys (Tab, Arrow keys) in real-time.
- **Pipeline Orchestration:** Parses commands separated by `|`, creates `os.Pipe` connections between them, and manages `sync.WaitGroup` for concurrent execution.
- **Signal Handling:** Detects special keys like `Ctrl+C` or `<Arrow Keys>` for history navigation.

### 2. Lexer (`pkg/lexer`)

- **`lexer.go`**: Converts raw input strings into a stream of typed tokens consumed by the parser.
- **Token Types:** Recognizes `WORD`, `PIPE` (`|`), `REDIRECT` (`>`, `>>`, `<`, `2>`, `2>>`), `SEMICOLON` (`;`), `AND` (`&&`), `OR` (`||`), and shell keywords (`if`, `then`, `else`, `fi`).
- **Quote Handling:** Correctly handles single quotes `'`, double quotes `"`, and backslash escapes `\`, so arguments with spaces are grouped correctly.

### 3. Parser & AST (`pkg/parser`, `pkg/ast`)

- **`parser.go`**: A recursive-descent parser that transforms the token stream into an Abstract Syntax Tree (AST).
- **`ast.go`**: Defines the node types that make up the tree:
  - `CommandNode` — a single command with its arguments.
  - `PipeNode` — connects two commands via a pipe (`|`).
  - `RedirectNode` — wraps a command with an I/O redirection.
  - `IfNode` — represents an `if / then / else / fi` conditional block.
  - `BlockNode` — a sequence of statements separated by `;`.
  - `BinaryNode` — logical operators `&&` (AND) and `||` (OR) between two commands.

### 4. Executor (`pkg/executor`)

- **`executor.go`**: Recursively walks the AST and executes each node:
  - **Pipes:** Spawns the left side in a goroutine writing to an `os.Pipe`, while the right side reads from it.
  - **Redirects:** Opens files with the correct flags (`O_TRUNC` / `O_APPEND`) and wires the appropriate file descriptor.
  - **Conditionals (`IfNode`):** Runs the condition block; if it succeeds (exit code 0), runs the `then` branch; otherwise the `else` branch.
  - **Logical operators (`BinaryNode`):** Short-circuits `&&` (runs right only on success) and `||` (runs right only on failure).

### 5. Command Registry (`pkg/commands`)

- **`registry.go`**: Maintains a map of built-in commands (`cd`, `exit`, `type`, `echo`, `pwd`, `ls`, `history`).
- **Trie-based Autocomplete**: Indexes $PATH executables and built-ins in a Trie, enabling Tab completion in O(k) time (prefix length), far faster than a linear O(n) scan over hundreds of binaries.

### 6. Terminal Control (`pkg/term`)

- **`term.go`**: Uses `syscall` to switch the terminal between **Canonical Mode** (buffered input) and **Raw Mode**.
- **Why?** Required to capture keystrokes like `Tab` (autocomplete) and `↑`/`↓` (history) immediately, without waiting for Enter.

### 7. History Management (`pkg/history`)

- **`history.go`**: Manages a thread-safe list of previous commands.
- **Persistence:** Reads from and writes to the file defined in `$HISTFILE` (like `.bash_history`).
- **Navigation:** `GetUpEntry()` and `GetDownEntry()` for scrolling through commands with the arrow keys.

---

## 🚀 Features

- **AST-Based Parsing:** Input is lexed into tokens and parsed into a full Abstract Syntax Tree before execution — enabling correct operator precedence and complex command composition.
- **Control Flow (`if / then / else / fi`):** Full support for conditional blocks, including optional `else` branches.
- **Logical Operators (`&&` / `||`):** Chain commands with short-circuit evaluation — `make && ./run` or `ping host || echo "unreachable"`.
- **Pipeline Support:** Run commands in parallel like `ls | grep .go | sort`.
- **I/O Redirection:** `>`, `>>`, `<`, `2>`, `2>>` for stdout, stdin, and stderr.
- **Tab Autocomplete:** Suggests commands from `$PATH` and built-ins (via Trie), and also completes **file and directory names** for arguments.
- **Command History:** Persistent history with `↑`/`↓` arrow key navigation.
- **Built-ins:** `cd`, `type`, `history`, `echo`, `pwd`, `ls`, `exit`.

---

## 🧠 The Execution Lifecycle Example
 
`ls /home | grep "go" > results.txt`
### 1. Lexing (Command → Tokens)
The Lexer breaks the raw input string into logical tokens:

`[WORD(ls), WORD(/home), PIPE(|), WORD(grep), WORD("go"), REDIRECT(>), WORD(results.txt), EOF]`

### 2. Parsing (Tokens → AST)
The Parser groups the tokens using strict precedence rules (Redirect > Pipe > Command) to build the Abstract Syntax Tree (AST):

```text
        Redirect (>)
           |
         Pipe (|)
        /        \
  ls /home    grep "go"
           \
        file: results.txt
```

### 3. Execution (AST → Registry)
The Executor recursively walks the AST from top to bottom, managing I/O streams and processes:

* **Redirect Node:** Opens `results.txt` and temporarily replaces the standard output stream.
* **Pipe Node:** Creates an `os.Pipe()`, running `ls` (writing) and `grep` (reading) concurrently.
* **Command Nodes:** Queries the **Command Registry** to see if `ls` and `grep` are built-in shell commands. Because they aren't, it resolves them via `$PATH` and executes them with the newly wired streams.


## 🛠 Usage

```bash
# Build the binary
go build -o myshell ./...

# Run the shell
./myshell
```


