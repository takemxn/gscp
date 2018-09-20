package scp

import (
	"fmt"
	"path/filepath"
	com "github.com/takemxn/gssh/common"
	"os"
	"io"
	"bufio"
	"errors"
)
func readExpect(r io.Reader, expect byte)(err error){
	b := make([]byte, 1)
	_, err = r.Read(b)
	if err != nil {
		return
	}
	switch b[0]{
	case expect:
		return
	case 1:
		// scp message
		br := bufio.NewReader(r)
		line, _, err := br.ReadLine()
		if err != nil {
			return err
		}
		return errors.New(string(line))
	default:
		err = fmt.Errorf("not expected receive:%v", b)
	}
	if b[0] != expect {
	}
	return
}

func (scp *Scp) sendFromRemote(file, user, host string, ech chan error) (r io.Reader, w io.WriteCloser, err error) {
	password := scp.Password
	if password == "" {
		password = scp.config.GetPassword(user, host, scp.Port)
	}
	ci := com.NewConnectInfo(user, host, scp.Port, password)
	conn, err := ci.Connect()
	if err != nil {
		scp.Printf("unable to create session: %s", err)
		return
	}
	s, err := conn.NewSession()
	if err != nil {
		return
	} else if scp.IsVerbose {
		scp.Println("Got sender session")
	}
	w, err = s.StdinPipe()
	if err != nil {
		return
	}
	r, err = s.StdoutPipe()
	if err != nil {
		return
	}
	e, err := s.StderrPipe()
	if err != nil {
		return
	}
	go io.Copy(scp.Stderr, e)
	go func(){
		defer func(){
			conn.Close()
			s.Close()
		}()
		remoteOpts := "-f"
		if scp.IsPreserve{
			remoteOpts += "p"
		}
		if scp.IsRecursive {
			remoteOpts += "r"
		}
		ech <- s.Run("/usr/bin/scp " + remoteOpts + " " + file)
	}()
	return
}
func (scp *Scp) sendFromLocal(srcFile string, reader io.Reader, writer io.Writer) (err error) {
	errPipe := scp.Stderr
	outPipe := scp.Stdout
	srcFileInfo, err := os.Stat(srcFile)
	if err != nil {
		return err
	}
	if scp.IsRecursive {
		if srcFileInfo.IsDir() {
			err = scp.processDir(reader, writer, srcFile, srcFileInfo, outPipe, errPipe)
			if err != nil {
				scp.Println( err.Error())
			}
		} else {
			err = scp.sendFile(reader, writer, srcFile, srcFileInfo, outPipe, errPipe)
			if err != nil {
				scp.Println( err.Error())
			}
		}
	} else {
		if srcFileInfo.IsDir() {
			return fmt.Errorf("%s: not a regular file", srcFile)
		} else {
			err = scp.sendFile(reader, writer, srcFile, srcFileInfo, outPipe, errPipe)
			if err != nil {
				//fmt.Println( err.Error())
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
	err = scp.sendEndDir(reader, writer, errPipe)
	return err
}

func (scp *Scp) sendEndDir(reader io.Reader, writer io.Writer, errPipe io.Writer) error {
	header := fmt.Sprintf("E\n")
	if scp.IsVerbose {
		scp.Printf("Sending end dir: %s", header)
	}
	_, err := writer.Write([]byte(header))
	return err
}

func (scp *Scp) sendDir(reader io.Reader, writer io.Writer, srcPath string, srcFileInfo os.FileInfo, errPipe io.Writer) error {
	mode := uint32(srcFileInfo.Mode().Perm())
	if scp.IsPreserve {
		// send time stamp
		mtime := srcFileInfo.ModTime()
		ftime := fmt.Sprintf("T%d 0 %d 0\n", mtime.Unix(), mtime.Unix())
		if scp.IsVerbose {
			scp.Printf("Sending File timestamp: %q\n", ftime)
		}
		_, err := writer.Write([]byte(ftime))
		if err != nil {
			return err
		}
		err = readExpect(reader, 0)
		if err != nil {
			return err
		}
	}
	header := fmt.Sprintf("D%04o 0 %s\n", mode, filepath.Base(srcPath))
	if scp.IsVerbose {
		scp.Printf("Sending Dir header : %s", header)
	}
	_, err := writer.Write([]byte(header))
	return err
}

func (scp *Scp) sendFile(reader io.Reader, writer io.Writer, srcPath string, srcFileInfo os.FileInfo, outPipe io.Writer, errPipe io.Writer) error {
	//single file
	mode := uint32(srcFileInfo.Mode().Perm())
	if scp.IsPreserve {
		// send time stamp
		mtime := srcFileInfo.ModTime()
		ftime := fmt.Sprintf("T%d 0 %d 0\n", mtime.Unix(), mtime.Unix())
		if scp.IsVerbose {
			scp.Printf("Sending File timestamp: %q\n", ftime)
		}
		_, err := writer.Write([]byte(ftime))
		if err != nil {
			return err
		}
		err = readExpect(reader, 0)
		if err != nil {
			return err
		}
	}
	fileReader, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer fileReader.Close()
	size := srcFileInfo.Size()
	header := fmt.Sprintf("C%04o %d %s\n", mode, size, filepath.Base(srcPath))
	if scp.IsVerbose {
		scp.Printf("Sending File header: %s", header)
	}
	pb := NewProgressBarTo(srcPath, size, outPipe)
	if !scp.IsQuiet {
		pb.Update(0)
		defer scp.Println()
	}
	_, err = writer.Write([]byte(header))
	if err != nil {
		return err
	}
	err = readExpect(reader, 0)
	if err != nil {
		return err
	}
	tot := int64(0)
	lastPercent := int64(0)
	var rb []byte
	for tot < size {
		rest := size - tot
		if rest < BUF_SIZE {
			rb = make([]byte, rest)
		}else{
			rb = make([]byte, BUF_SIZE)
		}
		n, err := fileReader.Read(rb)
		if err != nil {
			return err
		}
		wb := rb[:n]
		for i:=0;i < n;{
			wn, err := writer.Write(wb[i:])
			if err != nil {
				return err
			}
			i += wn
		}
		tot += int64(n)
		percent := (100 * tot) / size
		if percent > lastPercent {
			if !scp.IsQuiet {
				pb.Update(tot)
			}
		}
		lastPercent = percent
	}
	if scp.IsVerbose {
		scp.Println( "Sent file.")
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

	if err != nil {
		scp.Println( err.Error())
	}
	return err
}
