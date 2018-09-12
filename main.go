package main

import (
	"github.com/takemxn/gscp/scp"
	"os"
)

func main() {
	err := scp.ScpCli(os.Args)
	if err != nil {
		os.Exit(1)
	}
}
