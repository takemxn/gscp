// buffered-read-benchmark benchmarks the peformance of reading
// from /dev/zero on the server to a []byte on the client via io.Copy.
package main

import (
	"container/list"
	"errors"
	"flag"
	"fmt"
	com "github.com/takemxn/gssh/common"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"os"
	"os/user"
	"path"
	"regexp"
	"strconv"
	"strings"
)

type Item struct {
	username string
	hostname string
	filename string
	port     int
}

var (
	password        string
	username        string
	hostname        string
	port            int
	configPath      string
	command         string
	tFlag           bool
	vFlag           bool
	hFlag           bool
	NoPasswordError = errors.New("no password")
)

func init() {
	parseArg()
}

func main() {
	// Create client config
	config := &ssh.ClientConfig{
		User: com.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(com.Password),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	addr := fmt.Sprintf("%s:%d", com.Hostname, com.Port)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Printf("unable to connect: %s", err)
		return
	}
	defer conn.Close()
}
func parseArg() (err error) {
	port = 0
	args := os.Args
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	f.StringVar(&password, "p", "", "password")
	f.IntVar(&port, "P", 22, "port")
	f.StringVar(&configPath, "f", "", "password file path")
	f.BoolVar(&vFlag, "v", false, "show Version")
	f.BoolVar(&hFlag, "h", false, "show help")
	if err = f.Parse(args[1:]); err != nil {
		return
	}
	usage := func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", path.Base(os.Args[0]))
		f.PrintDefaults()
	}
	if vFlag {
		fmt.Println(path.Base(os.Args[0]), "version 0.9.0")
		os.Exit(0)
	}
	if hFlag {
		usage()
		os.Exit(1)
	}
	if f.NArg() <= 0 {
		usage()
		os.Exit(1)
	}

	// create source files
	srcList := list.New()
	for _, v := range f.Args() {
		re := regexp.MustCompile(`(.*)@?(.*):(.*)`)
		group := re.FindStringSubmatch(v)
		if len(group) == 0 {
			return fmt.Errorf("argument error")
		} else {
			uname := group[0]
			hname := group[1]
			fname := group[2]
			if len(hname) > 0 && len(uname) == 0 {
				// set current username if no username specified
				u, err := user.Current()
				if err != nil {
					return fmt.Errorf("argument error")
				}
				uname = u.Username
			}
			item := &Item{
				username: uname,
				hostname: hname,
				filename: fname,
				port:     port,
			}
			srcList.PushBack(item)
		}
	}
	rest := f.Arg(0)

	// Get hostname
	s := strings.Split(rest, ":")
	if len(s[0]) == 0 {
		return fmt.Errorf("hostname error")
	}
	hostname = s[0]

	// Get port number
	if len(s) >= 2 {
		port, err = strconv.Atoi(s[1])
	}

	switch {
	case password != "":
	default:
		err = com.ReadPasswords()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
		com.Password = com.GetPassword(com.Username, com.Hostname, com.Port)
		if len(com.Password) == 0 {
			return NoPasswordError
		}
	}

	// command
	command = strings.Join(f.Args()[1:], " ")

	return
}
func usage() {
	fmt.Fprintf(os.Stderr,
		`Usage: gscp [-f file] [-p password] [-P port] [[user@]host1:]file1 ... [[user@]host2:]file2
      if -p password is not set, $GSSH_PASSWORDFILE, $GSSH_PASSWORDS variable will be used.
      otherwise ~/.gssh file is used
        -p  password
        -P  port
        -f  password list filepath
        -v  Show version
        -h  Show help
`)
}
