package main

import (
	"fmt"
	"os"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("you need to enter token")
		return
	}
	value := os.Args[1]
	fmt.Println(value)
}
