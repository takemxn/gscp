package scp

import (
	"fmt"
	"path/filepath"
	com "github.com/takemxn/gssh/common"
	"os"
	"io"
	"sync"
)
var(
	wg sync.WaitGroup
)
func readExpect(r io.Reader, expect byte)(err error){
	b := make([]byte, 1)
	_, err = r.Read(b)
	if err != nil {
		return
	}
	if b[0] != expect {
		err = fmt.Errorf("not expected receive:%v", b)
	}
	return
}

func (scp *Scp) sendFromRemote(file, user, host string, reader io.Reader, writer io.WriteCloser) (err error) {
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
		defer wg.Done()
		wg.Add(1)
		for{
			buf := make([]byte, BUF_SIZE)
			n, err := r.Read(buf)
			if err != nil {
				return
			}
			_, err = writer.Write(buf[:n])
			if err != nil {
				return
			}
		}
	}()
	go func(){
		defer wg.Done()
		wg.Add(1)
		for{
			buf := make([]byte, BUF_SIZE)
			n, err := reader.Read(buf)
			if err != nil {
				return
			}
			_, err = w.Write(buf[:n])
			if err != nil {
				return
			}
		}
	}()
	go func(){
		defer wg.Done()
		wg.Add(1)
		io.Copy(scp.Stderr, e)
	}()
	remoteOpts := "-qf"
	if scp.IsPreserve{
		remoteOpts += "p"
	}
	if scp.IsRecursive {
		remoteOpts += "r"
	}
	err = s.Run("/usr/bin/scp " + remoteOpts + " " + file)
	if err != nil {
		return
	}
	return
}
func (scp *Scp) sendFromLocal(srcFile string, reader io.Reader, writer io.Writer) (err error) {
	errPipe := scp.Stderr
	outPipe := scp.Stdout
	srcFileInfo, err := os.Stat(srcFile)
	if err != nil {
		return err
	}
	/*
	err = readExpect(reader, 0)
	if err != nil {
		return err
	}
	*/
	if scp.IsRecursive {
		if srcFileInfo.IsDir() {
			err = scp.processDir(reader, writer, srcFile, srcFileInfo, outPipe, errPipe)
			if err != nil {
				fmt.Println( err.Error())
			}
		} else {
			err = scp.sendFile(reader, writer, srcFile, srcFileInfo, outPipe, errPipe)
			if err != nil {
				fmt.Println( err.Error())
			}
		}
	} else {
		if srcFileInfo.IsDir() {
			return
		} else {
			err = scp.sendFile(reader, writer, srcFile, srcFileInfo, outPipe, errPipe)
			if err != nil {
				fmt.Println( err.Error())
			}
		}
	}
	return
}
func (scp *Scp) processDir(reader io.Reader, writer io.Writer, srcFilePath string, srcFileInfo os.FileInfo, outPipe io.Writer, errPipe io.Writer) error {
	err := scp.sendDir(reader, writer, srcFilePath, srcFileInfo, errPipe)
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
			err = scp.processDir(reader, writer, filepath.Join(srcFilePath, fi.Name()), fi, outPipe, errPipe)
			if err != nil {
				return err
			}
		} else {
			err = scp.sendFile(reader, writer, filepath.Join(srcFilePath, fi.Name()), fi, outPipe, errPipe)
			if err != nil {
				return err
			}
		}
	}
	//TODO process errors
	err = scp.sendEndDir(reader, writer, errPipe)
	return err
}

func (scp *Scp) sendEndDir(reader io.Reader, writer io.Writer, errPipe io.Writer) error {
	header := fmt.Sprintf("E\n")
	if scp.IsVerbose {
		fmt.Fprintf(errPipe, "Sending end dir: %s", header)
	}
	_, err := writer.Write([]byte(header))
	return err
}

func (scp *Scp) sendDir(reader io.Reader, writer io.Writer, srcPath string, srcFileInfo os.FileInfo, errPipe io.Writer) error {
	mode := uint32(srcFileInfo.Mode().Perm())
	header := fmt.Sprintf("D%04o 0 %s\n", mode, filepath.Base(srcPath))
	if scp.IsVerbose {
		fmt.Fprintf(errPipe, "Sending Dir header : %s", header)
	}
	_, err := writer.Write([]byte(header))
	return err
}

func (scp *Scp) sendFile(reader io.Reader, writer io.Writer, srcPath string, srcFileInfo os.FileInfo, outPipe io.Writer, errPipe io.Writer) error {
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
	if !scp.IsQuiet {
		pb.Update(0)
	}
	_, err = writer.Write([]byte(header))
	if err != nil {
		return err
	}
	err = readExpect(reader, 0)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileReader)
	if err != nil {
		return err
	}
	if scp.IsVerbose {
		fmt.Println( "Sent file.")
	}
	err = sendByte(writer, 0)
	if err != nil {
		return err
	}
	err = readExpect(reader, 0)
	if err != nil {
		return err
	}
	if !scp.IsQuiet {
		pb.Update(size)
	}
	fmt.Fprintln(errPipe)

	if err != nil {
		fmt.Println( err.Error())
	}
	return err
}
func (scp *Scp) sendFrom(file string, reader io.Reader, writer io.WriteCloser) (err error) {
	wg := &sync.WaitGroup{}
	file, host, user, err := parseTarget(file)
	if err != nil {
		return
	}
	if host != "" {
		err = scp.sendFromRemote(file, user, host, reader, writer)
		if err != nil {
			return
		}
		wg.Wait()
		writer.Write([]byte{0x0A})
	} else {
		err = scp.sendFromLocal(file, reader, writer)
		if err != nil {
			return
		}
		wg.Wait()
	}
	return
}
