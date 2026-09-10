// Package main is a test protocol, not a runtime or host callback API.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	jsonata "github.com/openbindings/jsonata-runtime/go"
)

func main() {
	executor, err := jsonata.New(jsonata.Options{Timeout: 5 * time.Second})
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), 16<<20)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var request struct {
			ID, Expr, InputJSON string
			BindingsJSON        *string
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			panic(err)
		}
		var bindings []byte
		if request.BindingsJSON != nil {
			bindings = []byte(*request.BindingsJSON)
		}
		value, err := executor.Evaluate(context.Background(), request.Expr, []byte(request.InputJSON), bindings)
		result := map[string]any{"id": request.ID, "status": "json", "json": string(value)}
		if err != nil {
			result = map[string]any{"id": request.ID, "status": "failure", "error": err.Error()}
		}
		if err := encoder.Encode(result); err != nil {
			panic(err)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
