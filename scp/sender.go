package scp

import (
	"fmt"
	"path/filepath"
	com "github.com/takemxn/gssh/common"
	"os"
	"io"
)

func (scp *Scp) sendFromRemote(file, user, host string, iCH, oCH *Channel) (err error) {
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
		for{
			buf := make([]byte, BUF_SIZE)
			n, err := oCH.Read(buf)
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
		for{
			buf := make([]byte, BUF_SIZE)
			n, err := r.Read(buf)
			if err != nil {
				if err == io.EOF {
					iCH.Close()
				}
				return
			}
			_, err = iCH.Write(buf[:n])
			if err != nil {
				scp.Println("scp write error", err)
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
	err = s.Run("/usr/bin/scp " + remoteOpts + " " + file)
	if err != nil {
		return
	}
	s.Close()
	return
}
func (scp *Scp) sendFromLocal(srcFile string, iCH, oCH *Channel) (err error) {
	errPipe := scp.Stderr
	outPipe := scp.Stdout
	srcFileInfo, err := os.Stat(srcFile)
	if err != nil {
		fmt.Println( "Could not stat source file "+srcFile)
		return err
	}
	if scp.IsRecursive {
		if srcFileInfo.IsDir() {
			err = scp.processDir(iCH, oCH, srcFile, srcFileInfo, outPipe, errPipe)
			if err != nil {
				fmt.Println( err.Error())
			}
		} else {
			err = scp.sendFile(iCH, oCH, srcFile, srcFileInfo, outPipe, errPipe)
			if err != nil {
				fmt.Println( err.Error())
			}
		}
	} else {
		if srcFileInfo.IsDir() {
			return
		} else {
			err = scp.sendFile(iCH, oCH, srcFile, srcFileInfo, outPipe, errPipe)
			if err != nil {
				fmt.Println( err.Error())
			}
		}
	}
	return
}
func (scp *Scp) processDir(iCH io.Reader, oCH io.Writer, srcFilePath string, srcFileInfo os.FileInfo, outPipe io.Writer, errPipe io.Writer) error {
	err := scp.sendDir(iCH, oCH, srcFilePath, srcFileInfo, errPipe)
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
			err = scp.processDir(iCH, oCH, filepath.Join(srcFilePath, fi.Name()), fi, outPipe, errPipe)
			if err != nil {
				return err
			}
		} else {
			err = scp.sendFile(iCH, oCH, filepath.Join(srcFilePath, fi.Name()), fi, outPipe, errPipe)
			if err != nil {
				return err
			}
		}
	}
	//TODO process errors
	err = scp.sendEndDir(iCH, oCH, errPipe)
	return err
}

func (scp *Scp) sendEndDir(iCH io.Reader, oCH io.Writer, errPipe io.Writer) error {
	header := fmt.Sprintf("E\n")
	if scp.IsVerbose {
		fmt.Fprintf(errPipe, "Sending end dir: %s", header)
	}
	_, err := oCH.Write([]byte(header))
	return err
}

func (scp *Scp) sendDir(iCH io.Reader, oCH io.Writer, srcPath string, srcFileInfo os.FileInfo, errPipe io.Writer) error {
	mode := uint32(srcFileInfo.Mode().Perm())
	header := fmt.Sprintf("D%04o 0 %s\n", mode, filepath.Base(srcPath))
	if scp.IsVerbose {
		fmt.Fprintf(errPipe, "Sending Dir header : %s", header)
	}
	_, err := oCH.Write([]byte(header))
	return err
}

func (scp *Scp) sendFile(iCH io.Reader, oCH io.Writer, srcPath string, srcFileInfo os.FileInfo, outPipe io.Writer, errPipe io.Writer) error {
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
	_, err = oCH.Write([]byte(header))
	if err != nil {
		return err
	}
	//TODO buffering
	_, err = io.Copy(oCH, fileReader)
	if err != nil {
		return err
	}
	// terminate with null byte
	err = sendByte(oCH, 0)
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
func (scp *Scp) sendFrom(file string, iCH, oCH *Channel) (err error) {
	file, host, user, err := parseTarget(file)
	if err != nil {
		return
	}
	if host != "" {
		err = scp.sendFromRemote(file, user, host, iCH, oCH)
		if err != nil {
			return
		}
	} else {
		err = scp.sendFromLocal(file, iCH, oCH)
		if err != nil {
			return
		}
	}
	return
}
