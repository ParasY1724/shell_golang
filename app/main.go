package main

import (
	"fmt"
)


func main() {
	var cmd string
	for true {
		fmt.Print("$ ")
		fmt.Scan(&cmd)
		fmt.Println(cmd + ": command not found")
	}
}
