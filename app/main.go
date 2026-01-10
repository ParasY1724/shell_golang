package main

import (
	"fmt"
)


func main() {
	var cmd string
	for true {
		fmt.Print("$ ")
		fmt.Scan(&cmd)
		if cmd == "exit"{
			break
		}
		fmt.Println(cmd + ": command not found")
	}
}
