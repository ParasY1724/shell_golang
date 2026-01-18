package term

import (
	"syscall"
	"unsafe"
)

// EnableRawMode puts the terminal into raw mode to handle input character-by-character.
// fd: The file descriptor (usually 0 for os.Stdin).
func EnableRawMode(fd int) (*syscall.Termios, error) {
	var oldState syscall.Termios

	// 1. GET CURRENT SETTINGS
	// We use the SYS_IOCTL system call with the TCGETS (Terminal Control GET State) command.
	// This populates 'oldState' with the current terminal configuration.
	if _, _, err := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TCGETS),           // Command: Get current attributes
		uintptr(unsafe.Pointer(&oldState)), // Pointer to where to store the attributes
		0, 0, 0,
	); err != 0 {
		return nil, err
	}

	// 2. MODIFY SETTINGS
	// Create a copy so we can modify it while keeping 'oldState' safe to restore later.
	newState := oldState

	// Lflag (Local Mode Flags): Controls how the terminal handles input locally.
	// &^= is the "AND NOT" operator. It clears (turns off) the specified bits.
	// syscall.ICANON: Turns off Canonical mode.
	//    - ON: Input is buffered line-by-line (user must press Enter).
	//    - OFF: Input is available byte-by-byte immediately.
	// syscall.ECHO: Turns off Echo.
	//    - ON: Valid keys are printed to the screen automatically.
	//    - OFF: Keys are not printed. Our shell must print them manually (fmt.Print).
	newState.Lflag &^= syscall.ICANON | syscall.ECHO

	// Iflag (Input Mode Flags): Controls how input is processed before it reaches the program.
	// syscall.IXON: Turns off Software Flow Control (Ctrl+S to pause, Ctrl+Q to resume).
	// syscall.ICRNL: Turns off CR-to-NL translation.
	//    - ON: The Enter key sends carriage return '\r' (13), but terminal converts it to '\n' (10).
	//    - OFF: The Enter key is read literally as '\r' (13).
	//    (This is why we had to add case '\r' in your main.go!)
	newState.Iflag &^= syscall.IXON | syscall.ICRNL

	// Cc (Control Characters):
	// VMIN = 1: Read returns as soon as there is at least 1 byte available.
	// VTIME = 0: No timeout; wait indefinitely for that 1 byte.
	newState.Cc[syscall.VMIN] = 1
	newState.Cc[syscall.VTIME] = 0

	// 3. APPLY NEW SETTINGS
	// We use SYS_IOCTL again, but this time with TCSETS (Terminal Control SET State).
	if _, _, err := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TCSETS),           // Command: Set new attributes
		uintptr(unsafe.Pointer(&newState)), // Pointer to the new configuration
		0, 0, 0,
	); err != 0 {
		return nil, err
	}

	// Return the original state so we can restore it when the program exits.
	return &oldState, nil
}

// RestoreTerminal reverts the terminal to its original state.
// This is critical! If you don't call this before exiting, the user's terminal
// will stay in raw mode (no echo, weird Enter behavior) after your shell closes.
func RestoreTerminal(fd int, state *syscall.Termios) {
	syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TCSETS),           // Command: Set attributes back to oldState
		uintptr(unsafe.Pointer(state)),
		0, 0, 0,
	)
}