package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"regexp"
	"path/filepath"
	com "github.com/takemxn/gssh/common"
)

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
func copy(dst, src *Client) (err error) {
	if rFlag {
		err = src.Walk(src.Filename, func(path string, info os.FileInfo, e error)(err error){
			if e != nil {
				return e
			}
			if info.IsDir() && dst.IsDir(){
				dirname := filepath.Base(path)
				err = dst.Mkdir(dst.Filename + "/" + dirname, info.Mode())
				if err != nil {
					return err
				}
			}else if info.Mode().IsRegular() && dst.IsDir() {
				p := filepath.Base(path)
				err = copyFile(dst, dst.Filename + "/" + p, src, path) 
			}
			return
		})
	}else{
		if dst.IsDir(){
			p := filepath.Base(src.Filename)
			err = copyFile(dst, dst.Filename + "/" + p, src, src.Filename) 
		}else if src.Mode().IsRegular(){
			err = copyFile(dst, dst.Filename, src, src.Filename) 
		}else{
			log.Fatalf("not regular file")
		}
	}
	return
}
func main() {
	// get last location
	dst := locations[len(locations)-1]

	dcon, err := Connect(dst)
	if err != nil {
		log.Fatalf("dest connect error")
	}
	defer dcon.Close()

	for _, v := range locations[:len(locations)-1] {
		scon, err := Connect(v)
		if err != nil {
			log.Fatalf("src connect error")
		}
		defer scon.Close()

		copy(scon, dcon)
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
