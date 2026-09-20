package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {
	const BEARER_PREFIX = "Bearer "

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "you need to enter token")
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
			fmt.Fprintln(os.Stderr, "Argument is not Bearer prefix")
			return
		}

	default:
		fmt.Fprintln(os.Stderr, "User input more arguments!")
		return
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		fmt.Fprintln(os.Stderr, "Incorrect JWT token")
		return
	}
	header := parts[0]
	payload := parts[1]

	decodedHeader, err := decodeValue(header)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error handling JWT header:", err)
		return
	}

	decodedPayload, err := decodeValue(payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error handling JWT payload:", err)
		return
	}

	fmt.Println()

	headerString, err := formatJSON(decodedHeader)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error handling JWT header:", err)
		return
	}
	fmt.Println(headerString)

	payloadString, err := formatJSON(decodedPayload)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error handling JWT payload:", err)
		return
	}
	fmt.Println(payloadString)
}

func decodeValue(value string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(value)
}

func formatJSON(jsonData []byte) (string, error) {
	var buf bytes.Buffer
	err := json.Indent(&buf, jsonData, "", " ")
	return buf.String(), err
}
