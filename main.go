package main

import (
	"fmt"
	"github.com/takemxn/gscp/scp"
	"os"
)

func main() {
	err := scp.ScpCli(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
