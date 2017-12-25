package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/pkg/sftp"
	com "github.com/takemxn/gssh/common"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"os"
	"os/user"
	"path"
	"regexp"
)

type Loc struct {
	Username string
	Hostname string
	Filename string
	Port     int
	Password string
}
func (loc *Loc) IsRemote() bool {
	return len(loc.Hostname) != 0
}
type Client struct {
	*Loc
	sftp   *sftp.Client
	ssh    *ssh.Client
	os.FileInfo
}
func Connect(loc *Loc)(c *Client, err error){
	// Create client config
	config := &ssh.ClientConfig{
		User: loc.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(loc.Password),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	addr := fmt.Sprintf("%s:%d", loc.Hostname, loc.Port)
	ssh, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Printf("unable to connect: %s", err)
		return
	}

	sftp, err := sftp.NewClient(ssh, sftp.MaxPacket(*SIZE))
	if err != nil {
		ssh.Close()
		log.Fatalf("unable to start sftp subsytem: %v", err)
	}
	if loc.IsRemote() {
		c.FileInfo, err = sftp.Stat(loc.Filename)
		if err != nil {
			log.Fatalf("stat error")
		}
	}else{
		c.FileInfo, err = os.Stat(loc.Filename)
		if err != nil {
			log.Fatalf("stat error")
		}
	}
	c.ssh = ssh
	c.sftp = sftp
	c.Loc = loc
	return 
}
func (c *Client) Close() {
	if c.sftp != nil {
		c.sftp.Close()
		c.sftp = nil
	}
	if c.ssh != nil {
		c.ssh.Close()
		c.ssh = nil
	}
}
func (c *Client) IsDir() bool{
	return c.IsDir()
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
	rFlag           bool
	NoPasswordError = errors.New("no password")
	locations       []*Loc
	SIZE            = flag.Int("s", 1<<15, "set max packet size")
)

func init() {
	parseArg()
}

func copy(src, dst *Client) (err error) {
	if src.IsDir() && !rFlag {
		log.Fatalf("not regular file %v", src.Filename)
	}
	if src.IsDir() && rFlag {
	}
	if dst.IsRemote() {
		if dst.IsDir() {
			c := dst.sftp
			c.Create(dst.Filename + "/" + src.Filename)
		}
	}
	return
}
func main() {
	// get last location
	dst := locations[len(locations)-1]

	dConn, err := Connect(dst)
	if err != nil {
		log.Fatalf("dest connect error")
	}
	defer dConn.Close()

	for _, v := range locations[:len(locations)-1] {
		sConn, err := Connect(v)
		if err != nil {
			log.Fatalf("src connect error")
		}
		defer sConn.Close()

		err = copy(sConn, dConn)
		if err != nil {
			log.Fatalf("file copy error")
		}
	}
}
func parseArg() (err error) {
	f := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	f.StringVar(&password, "p", "", "password")
	f.IntVar(&port, "P", 22, "port")
	f.StringVar(&configPath, "f", "", "password file path")
	f.BoolVar(&vFlag, "v", false, "show Version")
	f.BoolVar(&hFlag, "h", false, "show help")
	f.BoolVar(&rFlag, "r", false, "ecursively copy entire directories.")
	if err = f.Parse(os.Args[1:]); err != nil {
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
			loc := &Loc{
				Username: uname,
				Hostname: hname,
				Filename: fname,
				Password: password,
				Port:     port,
			}
			// リモートロケーションかつパスワードが設定されていなければパスワードを入力を促す
			if loc.IsRemote() && len(password) == 0 {
				p, err := com.ReadPasswordFromTerminal()
				if err != nil {
					return err
				}
				loc.Password = p
			}
			locations = append(locations, loc)
		}
	}
	return
}
