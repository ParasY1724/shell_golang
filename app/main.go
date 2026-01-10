package main

import (
	"fmt"
)


func main() {
	var cmd string
	fmt.Print("$ ")
	fmt.Scan(&cmd)
	fmt.Println(cmd + ": command not found")
}
