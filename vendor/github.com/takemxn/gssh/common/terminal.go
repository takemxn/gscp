package common

import (
	"fmt"
	"os"
	"golang.org/x/crypto/ssh/terminal"
)
func ReadPasswordFromTerminal(user, host string)(passwd string, err error){
	fmt.Printf("%s@%s's password: ", user, host)
	p, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return
	}
	passwd = string(p)
	fmt.Println()
	return
}


