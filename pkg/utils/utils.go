package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func FindLeastPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	
	for _, s := range strs {
		for !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

func WriteHistory(cmdLine string){
	file, err := os.OpenFile(".go_shell_history",os.O_CREATE | os.O_APPEND | os.O_WRONLY , 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Writing History File : %s\n",err)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	_,err = writer.WriteString(cmdLine + "\n")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Writing History File : %s\n",err)
		return
	}
	writer.Flush()
}