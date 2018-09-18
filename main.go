package main

import (
	"os"
	"fmt"
	com "github.com/takemxn/gssh/common"
	"github.com/takemxn/gscp/scp"
)

func main() {
	scp := scp.NewScp(os.Stdin, os.Stdout, os.Stderr)
	err := scp.ParseFlags(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	config := com.NewConfig(scp.ConfigPath)
	err = config.ReadPasswords()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = scp.Exec(config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
