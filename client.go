package main
import (
	"os"
	"io"
	"github.com/pkg/sftp"
	"path/filepath"
	"golang.org/x/crypto/ssh"
)

type File struct{
	r *sftp.File
	l *os.File
}
type Client struct{
	*Loc
	sftp   *sftp.Client
	ssh    *ssh.Client
	os.FileInfo
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
func (f *Client) Create(path string)(file *File, err error){
	if f.sftp != nil {
		file.r, err = f.sftp.Create(path)
	}else{
		file.l, err = os.Create(path)
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
	return c.IsDir()
}
func (c *Client) Rel(basepath, path string) (rel string, err error){
	info, err := os.Lstat(basepath)
	if err != nil {
		return
	}
	if info.Mode().IsRegular(){
		rel = basepath
	}
	return
}
func (c *Client) Open(path string)(file *File, err error){
	return
}
func (file *File) Copy(src *File)(err error){
	if file.r != nil {
		if src.r != nil {
			_, err = io.Copy(file.r, src.r)
		}else if src.l != nil {
			_, err = io.Copy(file.r, src.l)
		}
	}else{
		if src.r != nil {
			_, err = io.Copy(file.l, src.r)
		}else if src.l != nil {
			_, err = io.Copy(file.l, src.l)
		}
	}
	return
}
func copyFile(dst *Client, dname string, src *Client, sname string) (err error){
	sfile, err := src.Open(sname)
	if err != nil {
		return
	}
	dfile, err := dst.Open(dname)
	if err != nil {
		return
	}
	dfile.Copy(sfile)
	return
}
