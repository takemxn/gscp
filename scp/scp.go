package scp

// thanks to this for inspiration ... https://gist.github.com/jedy/3357393

import (
	"errors"
	"github.com/laher/uggo"
	//	"golang.org/x/crypto/ssh"
	sshcon "github.com/takemxn/gssh/shared"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"strings"
)

const (
	VERSION = "0.1.0"
)
type ReadWriter struct {
	io.Reader
	io.WriteCloser
}
func NewReadWriter(r io.Reader, w io.WriteCloser) (rw *ReadWriter){
	rw = &ReadWriter{r, w}
	return
}
type Scp struct {
	Port              int
	IsRecursive       bool
	IsRemoteTo        bool
	IsRemoteFrom      bool
	IsQuiet           bool
	IsVerbose         bool
	dstHost          string
	dstUser          string
	dstFile          string
	args      []string
	Stdin io.Reader
	Stdout io.Writer
	Stderr io.Writer
	ses *ssh.Session
	ce chan error
}

func (scp *Scp) Name() string {
	return "scp"
}

//func Scp(call []string) error {
func (scp *Scp) ParseFlags(call []string, errPipe io.Writer) error {
	//fmt.Fprintf(errPipe, "Warning: this scp is incomplete and not currently working with all ssh servers\n")
	flagSet := uggo.NewFlagSetDefault("scp", "[options] [[user@]host1:]file1 [[user@]host2:]file2", VERSION)
	flagSet.BoolVar(&scp.IsRecursive, "r", false, "Recursive copy")
	flagSet.IntVar(&scp.Port, "P", 22, "Port number")
	flagSet.BoolVar(&scp.IsRemoteTo, "t", false, "Remote 'to' mode - not currently supported")
	flagSet.BoolVar(&scp.IsRemoteFrom, "f", false, "Remote 'from' mode - not currently supported")
	flagSet.BoolVar(&scp.IsQuiet, "q", false, "Quiet mode: disables the progress meter as well as warning and diagnostic messages")
	flagSet.BoolVar(&scp.IsVerbose, "v", false, "Verbose mode - output differs from normal scp")
	flagSet.StringVar(&sshcon.Password, "p", "", "password")
	flagSet.StringVar(&sshcon.ConfigPath, "F", "", "password file path")
	err, _ := flagSet.ParsePlus(call[1:])
	if err != nil {
		return err
	}
	if scp.IsRemoteTo || scp.IsRemoteFrom {
		return errors.New("This scp does NOT implement 'remote-remote scp'. Yet.")
	}
	args := flagSet.Args()
	if len(args) < 2 {
		flagSet.Usage()
		return errors.New("Not enough args")
	}
	scp.args = args
	return nil
}

//TODO: error for multiple ats or multiple colons
func parseTarget(target string) (string, string, string, error) {
	//treat windows drive refs as local
	if strings.Contains(target, ":\\") {
		if strings.Index(target, ":\\") == 1 {
			return target, "", "", nil
		}
	}
	if strings.Contains(target, ":") {
		//remote
		parts := strings.Split(target, ":")
		userHost := parts[0]
		file := parts[1]
		user := ""
		var host string
		if strings.Contains(userHost, "@") {
			uhParts := strings.Split(userHost, "@")
			user = uhParts[0]
			host = uhParts[1]
		} else {
			host = userHost
		}
		return file, host, user, nil
	} else {
		//local
		return target, "", "", nil
	}
}

func sendByte(w io.Writer, val byte) error {
	_, err := w.Write([]byte{val})
	return err
}

func NewScp(inPipe io.Reader, outPipe io.Writer, errPipe io.Writer) (*Scp) {
	return &Scp{Stdin:inPipe, Stdout:outPipe, Stderr:errPipe}
}
func ScpCli(args []string) error {
	scp := NewScp(os.Stdin, os.Stdout, os.Stderr)
	err := scp.ParseFlags(args, os.Stderr)
	if err != nil {
		return err
	}
	err = scp.Exec()
	return err
}
func (scp *Scp) Exec() (err error) {
	remoteCopy := false
	for _, v := range scp.args[:] {
		_, host, _, err := parseTarget(v)
		if err != nil {
			return err
		}
		if host != "" {
			remoteCopy = true
			break
		}
	}
	if remoteCopy == false {
		// local copy with scp
		// not supported yet
		return
	}
	//buf := make(chan []byte)
	scp.dstFile, scp.dstHost, scp.dstUser, err = parseTarget(scp.args[len(scp.args)-1])
	if err != nil {
		return err
	}
	rw, err := scp.OpenDst()
	if err != nil {
		return err
	}
	defer rw.Close()
	for _, v := range scp.args[0 : len(scp.args)-1] {
		err := scp.openSrc(v, rw)
		if err != nil {
			return err
		}
	}
	rw.Close()
	if scp.ses != nil {
		scp.ses.Wait()
	}else{
		<- scp.ce
	}
	return err
}
