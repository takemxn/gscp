package main
import (
	"os"
	"io"
	"github.com/pkg/sftp"
	"path/filepath"
	"golang.org/x/crypto/ssh"
)

type File interface {
	io.ReadWriteCloser
}
type Client struct{
	*Loc
	info os.FileInfo
	sftp   *sftp.Client
	ssh    *ssh.Client
}

func (f *Client) Mkdir(path string, mode os.FileMode) (err error){
	if f.sftp != nil {
		err = f.sftp.Mkdir(path)
		if err != nil {
			return err
		}
		err = f.sftp.Chmod(path, mode)
		if err != nil {
			return err
		}
	}else{
		err = os.Mkdir(path, mode)
	}
	return
}
func (c *Client) Create(path string)(file File, err error){
	if c.IsSftp() {
		file,err = c.sftp.Create(path)
	}else{
		file,err = os.Create(path)
	}
	return
}
func (f *Client) Walk(path string, wf filepath.WalkFunc)(err error){
	if f.sftp != nil {
		walker := f.sftp.Walk(path)
		for walker.Step() {
			err = wf(walker.Path(), walker.Stat(), walker.Err())
			if err != nil {
				return
			}
		}
	}else{
		err = filepath.Walk(path, wf)
	}
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
	info, err := os.Lstat(c.Path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
func (c *Client) IsSftp() bool {
	return c.sftp != nil
}
func (c *Client) Open(path string)(file File, err error){
	if c.IsSftp() {
		file, err = c.sftp.Open(path)
	}else{
		file, err = os.Open(path)
	}
	return
}
func copyFile(dst *Client, dname string, src *Client, sname string) (err error){
	s, err := src.Open(sname)
	if err != nil {
		return
	}
	defer s.Close()
	d, err := dst.Create(dname)
	if err != nil {
		return
	}
	defer d.Close()
	_, err = io.Copy(d, s)
	return
}
