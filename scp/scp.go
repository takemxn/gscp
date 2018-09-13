package scp

// thanks to this for inspiration ... https://gist.github.com/jedy/3357393

import (
	"errors"
	"github.com/laher/uggo"
	com "github.com/takemxn/gssh/common"
	"io"
	"os"
	"strings"
	"fmt"
)

const (
	VERSION = "0.1.0"
	BUF_SIZE = (4096)
)
type Channel struct {
	ch chan []byte
	buffer []byte
}
func NewChannel() *Channel{
	ch := &Channel{}
	ch.ch = make(chan []byte, 1)
	return ch
}
func (ch *Channel) Write(p []byte) (n int, err error){
	ch.ch <- p
	return len(p), nil
}
func (ch *Channel) Read(p []byte) (n int, err error){
	if len(ch.buffer) > 0{
		n := copy(p, ch.buffer)
		ch.buffer = ch.buffer[n:]
		return n, nil
	}else{
		b, ok := <- ch.ch
		if ok {
			n := copy(p, b)
			ch.buffer = b[n:]
			return n, nil
		}else{
			return 0, io.EOF
		}
	}
	return
}
func (ch *Channel) Close() (err error){
	select {
	case <- ch.ch:
	default:
		close(ch.ch)
	}
	return nil
}
type Scp struct {
	Port              int
	IsRecursive       bool
	IsRemoteTo        bool
	IsRemoteFrom      bool
	IsQuiet           bool
	IsVerbose         bool
	IsPreserve				bool
	dstHost          string
	dstUser          string
	dstFile          string
	args      []string
	Stdin io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Password string
	ConfigPath string
	config *com.Config
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
	flagSet.BoolVar(&scp.IsPreserve, "p", false, "Preserve mode from the original file")
	flagSet.StringVar(&scp.Password, "w", "", "password")
	flagSet.StringVar(&scp.ConfigPath, "F", "", "password file path")
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
	if scp.Password == "" {
		config := com.NewConfig(scp.ConfigPath)
		err = config.ReadPasswords()
		if err != nil {
			return err
		}
		scp.config = config
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
	rCh := make(chan error, 1)
	in, out, err := scp.openReceiver(rCh)
	if err != nil {
		return err
	}
	sCh := make(chan error, 1)
	go func(){
		for _, v := range scp.args[0 : len(scp.args)-1] {
			err := scp.sendFrom(v, in, out)
			if err != nil {
				sCh <- err
			}
		}
		out.Close()
		sCh <-nil
	}()
	err = <-rCh
	if err != nil {
		return err
	}
	err = <-sCh
	if err != nil {
		return err
	}
	return
}
func (scp *Scp) Printf(format string, args ...interface{}){
	fmt.Fprintf(scp.Stderr, format, args...)
}
func (scp *Scp) Println(args ...interface{}){
	fmt.Fprintln(scp.Stderr, args...)
}
