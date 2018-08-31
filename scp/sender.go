package scp

import (
	"fmt"
	"path/filepath"
	sshcon "github.com/takemxn/gssh/shared"
	"log"
	"os"
	"io"
)

func (scp *Scp) sendFromRemote(file, user, host string, rw *ReadWriter) (err error) {
	conn, err := sshcon.Connect2(user, host, scp.Port)
	if err != nil {
		log.Printf("unable to create session: %s", err)
		return
	}
	s, err := conn.NewSession()
	if err != nil {
		return
	} else if scp.IsVerbose {
		log.Println( "Got session")
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
	go io.Copy(w, rw)
	go io.Copy(rw, r)
	remoteOpts := "-f"
	if scp.IsQuiet {
		remoteOpts += "q"
	}
	if scp.IsRecursive {
		remoteOpts += "r"
	}
	//TODO should this path (/usr/bin/scp) be configurable?
	err = s.Start("/usr/bin/scp " + remoteOpts + " " + file)
	if err != nil {
		log.Println( "Failed to run remote scp: ",err)
	}
	s.Wait()
	return
}
func (scp *Scp) sendFromLocal(srcFile string, w io.Writer) (err error) {
	errPipe := scp.Stderr
	outPipe := scp.Stdout
	srcFileInfo, err := os.Stat(srcFile)
	if err != nil {
		log.Println( "Could not stat source file "+srcFile)
		return err
	}
	procWriter := w
	if scp.IsRecursive {
		if srcFileInfo.IsDir() {
			err = scp.processDir(procWriter, srcFile, srcFileInfo, outPipe, errPipe)
			if err != nil {
				log.Println( err.Error())
			}
		} else {
			err = scp.sendFile(procWriter, srcFile, srcFileInfo, outPipe, errPipe)
			if err != nil {
				log.Println( err.Error())
			}
		}
	} else {
		if srcFileInfo.IsDir() {
			return
		} else {
			err = scp.sendFile(procWriter, srcFile, srcFileInfo, outPipe, errPipe)
			if err != nil {
				log.Println( err.Error())
			}
		}
	}
	return
}
type src struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
}
func (s *src) StdoutPipe() (io.Reader, error)     { return s.stdout, nil }
func (s *src) StdinPipe() (io.WriteCloser, error) { return s.stdin, nil }
func (s *src) StderrPipe() (io.Reader, error)     { return s.stdout, nil }
func (s *src) Start(cmd string) error               { return nil }
func (s *src) Wait() error               { return nil }
func (s *src) Close() error                       { return nil }

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
		log.Println( "Sent file plus null-byte.")
	}
	pb.Update(size)
	fmt.Fprintln(errPipe)

	if err != nil {
		log.Println( err.Error())
	}
	return err
}
func (scp *Scp) sendFrom(file string, rw *ReadWriter) (err error) {
	file, host, user, err := parseTarget(file)
	if err != nil {
		return
	}
	if host != "" {
		err = scp.sendFromRemote(file, user, host, rw)
		if err != nil {
			return
		}
	} else {
		err = scp.sendFromLocal(file, rw)
		if err != nil {
			return
		}
	}
	return
}
