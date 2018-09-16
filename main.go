package main

import (
	"github.com/takemxn/gscp/scp"
	"os"
	"fmt"
)

func main() {
	err := scp.ScpCli(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
