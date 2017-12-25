package main

import (
	"errors"
	"flag"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	com "gssh/common"
)

var (
	command    string
	passEnv    bool
	tFlag      bool
	vFlag      bool
	hFlag      bool
	NoPasswordError = errors.New("no password")
)

type Session struct {
	ssh.Session
}

func main() {
	err := parseArg()
	if err == NoPasswordError {
		com.Password, err = com.ReadPasswordFromTerminal()
	}
	if err != nil {
		usage()
		os.Exit(1)
	}
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

	// Create a session
	session, err := conn.NewSession()
	if err != nil {
		log.Printf("unable to create session: %s", err)
		return
	}
	defer session.Close()

	// Terminal file descpriter?
	s := &Session{*session}
	if terminal.IsTerminal(int(os.Stdin.Fd())) {
		err = s.remoteShell()
	} else {
		err = s.remoteExec()
	}

	// Exit status
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			os.Exit(exitErr.ExitStatus())
		} else {
			log.Printf("ExitError:%v", err)
			os.Exit(1)
		}
	}
	// Succeeded
	os.Exit(0)
}
func parseArg() (err error) {

	com.Username, com.Password, com.Hostname = "", "", ""
	com.Port = 0

	args := os.Args

	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	f.StringVar(&com.Password, "p", "", "password")
	f.StringVar(&com.ConfigPath, "f", "", "password file path")
	f.BoolVar(&passEnv, "e", false, "passing to pty")
	f.BoolVar(&tFlag, "t", false, "Force pseudo-tty allocation")
	f.BoolVar(&vFlag, "v", false, "Display Version")
	f.BoolVar(&hFlag, "h", false, "help")
	if err = f.Parse(args[1:]); err != nil {
		return
	}
	if vFlag {
		fmt.Println(path.Base(os.Args[0]), "version 0.9.0")
		os.Exit(0)
	}
	if hFlag {
		help()
		os.Exit(1)
	}
	if f.NArg() <= 0 {
		return fmt.Errorf("too few argument")
	}

	// Get user name
	rest := f.Arg(0)
	if strings.Contains(f.Arg(0), "@") {
		s := strings.Split(f.Arg(0), "@")
		if len(s[0]) == 0 {
			return fmt.Errorf("user name error")
		}
		com.Username = s[0]
		rest = s[1]
	}else if com.Username == "" {
		u, _ := user.Current()
		com.Username = u.Username
	}


	// Get hostname
	s := strings.Split(rest, ":")
	if len(s[0]) == 0 {
		return fmt.Errorf("hostname error")
	}
	com.Hostname = s[0]

	// Get port number
	if len(s) >= 2 {
		com.Port, err = strconv.Atoi(s[1])
	}
	// Connect to ssh server
	if com.Port == 0 {
		com.Port = 22
	}

	switch {
	case com.Password != "":
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
		`Usage: gssh [-t] [-p password] [-f file] [-v] [user@]hostname[:port] [command]
      if -p password is not set, $GSSH_PASSWORDFILE, $GSSH_PASSWORDS variable will be used.
      otherwise ~/.gssh file is used
        -p  password
        -f  password list filepath
        -t  Force pseudo-tty allocation.
        -v  Show version
        -h  Show help
`)
}
func help() {
	fmt.Fprintf(os.Stderr,
		`Usage: gssh [-t] [-p password] [-f file] [-v] [user@]hostname[:port] [command]
      if -p password is not set, $GSSH_PASSWORDFILE, $GSSH_PASSWORDS variable will be used.
      otherwise ~/.gssh file is used
        -p  password
        -f  password list filepath
        -t  Force pseudo-tty allocation.
        -v  Show version
        -h  help
      for example:
        1) $ cat gssh.conf
            [passwords]
            user1 = password1
            user2@hostname = password2
           $ export GSSH_PASSWORDFILE=gssh.conf
           $ gssh user1@hostname
           $ gssh user2@hostname
        2) $ export GSSH_PASSWORDS="user1=password1 user2@hostname=password2"
           $ gssh user1@hostname
           $ gssh user2@hostname
`)
}

func (s *Session) requestWindowChange(w, h int) (err error) {
	// RFC 4254 Section 6.7.
	req := struct {
		Columns uint32
		Rows    uint32
		Width   uint32
		Height  uint32
	}{
		Columns: uint32(w),
		Rows:    uint32(h),
		Width:   uint32(w * 8),
		Height:  uint32(h * 8),
	}
	ok, err := s.SendRequest("window-change", false, ssh.Marshal(&req))
	if err == nil && !ok {
		err = errors.New("ssh: window-change failed")
	}
	return err
}
func (s *Session) requestEnv(name, value string) (err error) {
	// RFC 4254 Section 6.4.
	req := struct {
		name  string
		value string
	}{
		name:  name,
		value: value,
	}
	fmt.Printf("name:[%s], value[%s]", name, value)
	ok, err := s.SendRequest("env", false, ssh.Marshal(&req))
	if err == nil && !ok {
		err = errors.New("ssh: env failed")
	}
	return err
}

func (s *Session) passEnvPty() (err error) {
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		err = s.requestEnv(pair[0], pair[1])
		if err != nil {
			break
		}
	}
	return
}

func (s *Session) remoteShell() (err error) {
	exit := make(chan struct{}, 1)
	wg := &sync.WaitGroup{}

	stdin, _ := s.StdinPipe()
	stdout, _ := s.StdoutPipe()
	stderr, _ := s.StderrPipe()

	go io.Copy(stdin, os.Stdin)

	go func() {
		wg.Add(1)
		defer wg.Done()
		io.Copy(os.Stdout, stdout)
	}()
	go func() {
		wg.Add(1)
		defer wg.Done()
		io.Copy(os.Stderr, stderr)
	}()

	if command != "" {
		if tFlag {
			// Forced request pseudo terminal
			w, h, err := terminal.GetSize(int(os.Stdin.Fd()))
			if err != nil {
				log.Printf("request for terminal window size failed: %s", err)
				return err
			}
			// Set up terminal modes
			modes := ssh.TerminalModes{
				ssh.ECHO: 0,
			}
			if err := s.RequestPty("xterm", h, w, modes); err != nil {
				log.Printf("request for pseudo terminal failed: %s", err)
				return err
			}
		}
		// Exec Command
		if err = s.Start(command); err != nil {
			log.Printf("failed to start shell: %s", err)
			return err
		}
	} else {
		// For pipe
		fd := int(os.Stdin.Fd())

		// Make terminal into raw mode
		oldState, _ := terminal.MakeRaw(int(os.Stdin.Fd()))
		defer terminal.Restore(fd, oldState)

		// Request pseudo terminal
		w, h, err := terminal.GetSize(fd)
		if err != nil {
			log.Printf("request for terminal window size failed: %s", err)
			return err
		}

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH)
		go func() {
			for {
				select {
				case v := <-sig:
					switch v {
					case syscall.SIGINT, syscall.SIGTERM:
						return
					case syscall.SIGWINCH:
						if w, h, err := terminal.GetSize(fd); err == nil {
							s.requestWindowChange(w, h)
						}
					}
				case <-exit:
					return
				}
			}
		}()

		// Set up terminal modes
		modes := ssh.TerminalModes{
			ssh.ECHO: 1,
		}
		if err := s.RequestPty("xterm", h, w, modes); err != nil {
			log.Printf("request for pseudo terminal failed: %s", err)
			return err
		}
		// Start Remote Shell
		if err = s.Shell(); err != nil {
			log.Printf("failed to start shell: %s", err)
			return err
		}
	}
	err = s.Wait()
	wg.Wait()
	close(exit)
	return
}

func (s *Session) remoteExec() (err error) {
	stdin, _ := s.StdinPipe()
	stdout, _ := s.StdoutPipe()
	stderr, _ := s.StderrPipe()

	wg := &sync.WaitGroup{}
	go func() {
		wg.Add(1)
		defer wg.Done()
		io.Copy(stdin, os.Stdin)
		stdin.Close()
	}()

	go func() {
		wg.Add(1)
		defer wg.Done()
		io.Copy(os.Stdout, stdout)
	}()
	go func() {
		wg.Add(1)
		defer wg.Done()
		io.Copy(os.Stderr, stderr)
	}()

	if tFlag {
		// Forced request pseudo terminal

		// Set up terminal modes
		modes := ssh.TerminalModes{
			ssh.ECHO: 0,
		}
		if err := s.RequestPty("xterm", 80, 40, modes); err != nil {
			log.Printf("request for pseudo terminal failed: %s", err)
			return err
		}
	}
	if command != "" {
		// Exec Command
		if err = s.Start(command); err != nil {
			log.Printf("failed to start shell: %s", err)
			return err
		}
	} else {
		// Start Remote Shell
		if err = s.Shell(); err != nil {
			log.Printf("failed to start shell: %s", err)
			return err
		}
	}

	err = s.Wait()
	wg.Wait()
	return
}
