package scp

import (
	"fmt"
	"path/filepath"
	com "github.com/takemxn/gssh/common"
	"os"
	"io"
)

func (scp *Scp) sendFromRemote(file, user, host string, in, out *Channel) (err error) {
	password := scp.Password
	if password == "" {
		password = scp.config.GetPassword(user, host, scp.Port)
	}
	ci := com.NewConnectInfo(user, host, scp.Port, password)
	conn, err := ci.Connect()
	if err != nil {
		fmt.Printf("unable to create session: %s", err)
		return
	}
	s, err := conn.NewSession()
	if err != nil {
		return
	} else if scp.IsVerbose {
		fmt.Fprintln(scp.Stderr, "Got sender session")
	}
	defer s.Close()
	w, err := s.StdinPipe()
	if err != nil {
		return
	}
	r, err := s.StdoutPipe()
	if err != nil {
		return
	}
	e, err := s.StderrPipe()
	if err != nil {
		return
	}
	go func(){
		//io.Copy(w, out)
		for{
			buf := make([]byte, BUF_SIZE)
			n, err := out.Read(buf)
			if err != nil {
				if err == io.EOF{
					w.Close()
				}
				return
			}
			_, err = w.Write(buf[:n])
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}()
	go func(){
		//io.Copy(in, r)
		for{
			buf := make([]byte, BUF_SIZE)
			n, err := r.Read(buf)
			if err != nil {
				if err == io.EOF {
					in.Close()
				}
				return
			}
			_, err = in.Write(buf[:n])
			if err != nil {
				fmt.Println(err)
				fmt.Println("scp write error", err)
				return
			}
		}
	}()
	go io.Copy(scp.Stderr, e)
	remoteOpts := "-pf"
	if scp.IsQuiet {
		remoteOpts += "q"
	}
	if scp.IsRecursive {
		remoteOpts += "r"
	}
	//TODO should this path (/usr/bin/scp) be configurable?
	err = s.Run("/usr/bin/scp " + remoteOpts + " " + file)
	if err != nil {
		return
	}
	s.Close()
	return
}
func (scp *Scp) sendFromLocal(srcFile string, in, out *Channel) (err error) {
	errPipe := scp.Stderr
	outPipe := scp.Stdout
	srcFileInfo, err := os.Stat(srcFile)
	if err != nil {
		fmt.Println( "Could not stat source file "+srcFile)
		return err
	}
	procWriter := out
	if scp.IsRecursive {
		if srcFileInfo.IsDir() {
			err = scp.processDir(procWriter, srcFile, srcFileInfo, outPipe, errPipe)
			if err != nil {
				fmt.Println( err.Error())
			}
		} else {
			err = scp.sendFile(procWriter, srcFile, srcFileInfo, outPipe, errPipe)
			if err != nil {
				fmt.Println( err.Error())
			}
		}
	} else {
		if srcFileInfo.IsDir() {
			return
		} else {
			err = scp.sendFile(procWriter, srcFile, srcFileInfo, outPipe, errPipe)
			if err != nil {
				fmt.Println( err.Error())
			}
		}
	}
	return
}
func (scp *Scp) processDir(procWriter io.Writer, srcFilePath string, srcFileInfo os.FileInfo, outPipe io.Writer, errPipe io.Writer) error {
	err := scp.sendDir(procWriter, srcFilePath, srcFileInfo, errPipe)
	if err != nil {
		return err
	}
	dir, err := os.Open(srcFilePath)
	if err != nil {
		return err
	}
	fis, err := dir.Readdir(0)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		if fi.IsDir() {
			err = scp.processDir(procWriter, filepath.Join(srcFilePath, fi.Name()), fi, outPipe, errPipe)
			if err != nil {
				return err
			}
		} else {
			err = scp.sendFile(procWriter, filepath.Join(srcFilePath, fi.Name()), fi, outPipe, errPipe)
			if err != nil {
				return err
			}
		}
	}
	//TODO process errors
	err = scp.sendEndDir(procWriter, errPipe)
	return err
}

func (scp *Scp) sendEndDir(procWriter io.Writer, errPipe io.Writer) error {
	header := fmt.Sprintf("E\n")
	if scp.IsVerbose {
		fmt.Fprintf(errPipe, "Sending end dir: %s", header)
	}
	_, err := procWriter.Write([]byte(header))
	return err
}

func (scp *Scp) sendDir(procWriter io.Writer, srcPath string, srcFileInfo os.FileInfo, errPipe io.Writer) error {
	mode := uint32(srcFileInfo.Mode().Perm())
	header := fmt.Sprintf("D%04o 0 %s\n", mode, filepath.Base(srcPath))
	if scp.IsVerbose {
		fmt.Fprintf(errPipe, "Sending Dir header : %s", header)
	}
	_, err := procWriter.Write([]byte(header))
	return err
}

func (scp *Scp) sendFile(procWriter io.Writer, srcPath string, srcFileInfo os.FileInfo, outPipe io.Writer, errPipe io.Writer) error {
	//single file
	mode := uint32(srcFileInfo.Mode().Perm())
	fileReader, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer fileReader.Close()
	size := srcFileInfo.Size()
	header := fmt.Sprintf("C%04o %d %s\n", mode, size, filepath.Base(srcPath))
	if scp.IsVerbose {
		fmt.Fprintf(errPipe, "Sending File header: %s", header)
	}
	pb := NewProgressBarTo(srcPath, size, outPipe)
	pb.Update(0)
	_, err = procWriter.Write([]byte(header))
	if err != nil {
		return err
	}
	//TODO buffering
	_, err = io.Copy(procWriter, fileReader)
	if err != nil {
		return err
	}
	// terminate with null byte
	err = sendByte(procWriter, 0)
	if err != nil {
		return err
	}

	err = fileReader.Close()
	if scp.IsVerbose {
		fmt.Println( "Sent file plus null-byte.")
	}
	pb.Update(size)
	fmt.Fprintln(errPipe)

	if err != nil {
		fmt.Println( err.Error())
	}
	return err
}
func (scp *Scp) sendFrom(file string, in, out *Channel) (err error) {
	file, host, user, err := parseTarget(file)
	if err != nil {
		return
	}
	if host != "" {
		err = scp.sendFromRemote(file, user, host, in, out)
		if err != nil {
			return
		}
	} else {
		err = scp.sendFromLocal(file, in, out)
		if err != nil {
			return
		}
	}
	return
}
