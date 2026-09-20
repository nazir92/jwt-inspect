package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

func main() {
	const BEARER_PREFIX = "Bearer "

	if len(os.Args) < 2 {
		fmt.Println("you need to enter token")
		return
	}
	argsCount := len(os.Args)
	argFirstValue := os.Args[1]
	token := ""
	switch argsCount {
	case 2:
		if after, ok := strings.CutPrefix(argFirstValue, BEARER_PREFIX); ok {
			token = after
		} else {
			token = argFirstValue
		}
	case 3:
		if os.Args[1] == strings.TrimSpace(BEARER_PREFIX) {
			token = os.Args[2]
		} else {
			fmt.Println("Argument is not Bearer prefix")
			return
		}

	default:
		fmt.Println("User input more arguments!")
		return
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		fmt.Println("Incorrect JWT token")
		return
	}
	header := parts[0]
	payload := parts[1]

	decodedHeader, err := base64.RawURLEncoding.DecodeString(header)
	if err != nil {
		fmt.Println("Error happened when parsing header")
		return
	}

	decodedPayload, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		fmt.Println("Error happend when parsing payload")
		return
	}

	fmt.Println(string(decodedHeader))
	fmt.Println(string(decodedPayload))
}
