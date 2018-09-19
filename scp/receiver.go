package scp

import (
	"fmt"
	"errors"
	com "github.com/takemxn/gssh/common"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"bufio"
)
type FileSet struct{
	ftype string
	mode os.FileMode
	size int64
	atime time.Time
	mtime time.Time
	filename string
}
func (scp *Scp) openLocalReceiver(rd io.Reader, cw io.Writer) (err error) {
	dstFile := scp.dstFile
	errPipe := scp.Stderr
	dstFileInfo, e := os.Stat(dstFile)
	dstDir := dstFile
	dstName := ""
	dstFileNotExist := false
	if e != nil {
		if !os.IsNotExist(e) {
			return e
		}
		dstDir = filepath.Dir(dstFile)
		dstName = filepath.Base(dstFile)
		dstFileNotExist = true
	} else if dstFileInfo.IsDir() {
		dstDir = dstFile
	} else if dstFileInfo.Mode().IsRegular() {
		dstDir = filepath.Dir(dstFile)
		dstName = dstFileInfo.Name()
	}else{
		return errors.New("spcified file was not dir or regular file!!")
	}
	defer func(){
		if scp.IsVerbose {
			scp.Println("local receiver end")
		}
	}()
	if scp.IsVerbose {
		scp.Println("Sending null byte")
	}
	err = sendByte(cw, 0)
	if err != nil {
		return
	}
	fs := new(FileSet)
	for {
		b := make([]byte, 1)
		n, err := rd.Read(b)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if n == 0 {
			return err
		}
		cmd := b[0]
		if scp.IsVerbose {
			scp.Printf("cmd : [%s](%02x)\n", string(cmd), cmd)
		}
		switch cmd {
		case 0x0:
			//continue
			if scp.IsVerbose {
				scp.Printf("Received OK \n")
			}
		case 0x1:
			// scp message
			br := bufio.NewReader(rd)
			line, _, err := br.ReadLine()
			if err != nil {
				return err
			}
			return errors.New(string(line))
		case 'E':
			//E command: go back out of dir
			dstDir = filepath.Dir(dstDir)
			if scp.IsVerbose {
				scp.Printf("Received End-Dir\n")
			}
			err = sendByte(cw, 0)
			if err != nil {
				scp.Println("Write error: %s", err.Error())
				return err
			}
		case 0xA:
			//0xA command: end?
			if scp.IsVerbose {
				scp.Printf("Received All-done\n")
			}
		case 'D':
			parts, err := scp.parseCmdLine(rd)
			if err != nil {
				return err
			}
			fs.mode, fs.size, fs.filename, err = scp.parseCmd(parts)
			if err != nil {
				return err
			}
			if !scp.IsRecursive {
				err = fmt.Errorf("scp: %q/%q is not aregular file", dstDir, fs.filename)
				break
			}
			fileMode := os.FileMode(uint32(fs.mode))
			if dstFileNotExist {
				if scp.IsVerbose {
					scp.Printf("makdir %q\n", dstDir)
				}
				err = os.Mkdir(dstDir, fileMode)
				if err != nil {
					return err
				}
			}else if !dstFileInfo.IsDir(){
				err = fmt.Errorf("scp: %q: Not a directory", dstFile)
				return err
			}
			//D command (directory)
			thisDstFile := filepath.Join(dstDir, fs.filename)
			err = os.MkdirAll(thisDstFile, fileMode)
			if err != nil {
				return err
			}
			if scp.IsPreserve {
				if err := os.Chtimes(thisDstFile, fs.atime, fs.mtime); err != nil {
					return err
				}
			}
			dstDir = thisDstFile
			err = sendByte(cw, 0)
			if err != nil {
				return err
			}
		case 'C':
			parts, err := scp.parseCmdLine(rd)
			if err != nil {
				return err
			}
			fs.mode, fs.size, fs.filename, err = scp.parseCmd(parts)
			if err != nil {
				return err
			}
			err = sendByte(cw, 0)
			if err != nil {
				scp.Println("Write error: %s", err.Error())
				return err
			}
			err = scp.receiveFile(rd, cw, dstDir, dstName, fs, errPipe)
			if err != nil {
				return err
			}
			err = sendByte(cw, 0)
			if err != nil {
				scp.Println("Write error: %s", err.Error())
				return err
			}
		case 'T':
			parts, err := scp.parseCmdLine(rd)
			if err != nil {
				return err
			}
			// modification time
			t, err := strconv.ParseUint(parts[0], 10, 64)
			if err != nil {
				return err
			}
			fs.mtime = time.Unix(int64(t), 0)
			// access time
			t, err = strconv.ParseUint(parts[2], 10, 64)
			if err != nil {
				return err
			}
			fs.atime = time.Unix(int64(t), 0)
			err = sendByte(cw, 0)
			if err != nil {
				scp.Println("Write error: %s", err.Error())
				return err
			}
		default :
			if scp.IsVerbose{
				err = fmt.Errorf("scp: Command '%x' NOT implemented\n", cmd)
			}
			return err
		}
	}
	return
}
func (scp *Scp) openRemoteReceiver(rCh chan error) (r io.Reader, w io.WriteCloser, err error) {
	password := scp.Password
	if password == "" {
		password = scp.config.GetPassword(scp.dstUser,  scp.dstHost, scp.Port)
	}
	ci := com.NewConnectInfo(scp.dstUser, scp.dstHost, scp.Port, password)
	conn, err := ci.Connect()
	if err != nil {
		scp.Printf("unable to create session: %s", err)
		return
	}
	s, err := conn.NewSession()
	if err != nil {
		return
	} else if scp.IsVerbose {
		scp.Println("Got receiver session")
	}
	w, err = s.StdinPipe()
	if err != nil {
		return
	}
	r, err = s.StdoutPipe()
	if err != nil {
		return
	}
	remoteOpts := "-t"
	if scp.IsPreserve{
		remoteOpts += "p"
	}
	if scp.IsRecursive {
		remoteOpts += "r"
	}
	go func(){
		defer s.Close()
		rCh <- s.Run("/usr/bin/scp " + remoteOpts + " " + scp.dstFile)
	}()
	return
}
func (scp *Scp)parseCmd(cmdStr []string) (mode os.FileMode, size int64, filename string, err error){
	m, err := strconv.ParseInt(cmdStr[0], 8, 32)
	if err != nil {
		return
	}
	mode = os.FileMode(m)
	sizeUint, err := strconv.ParseUint(cmdStr[1], 10, 64)
	size = int64(sizeUint)
	if err != nil {
		return
	}
	filename = cmdStr[2]
	if scp.IsVerbose {
		scp.Printf("Mode: %03o, size: %d, filename: %s\n", mode, size, filename)
	}
	return
}
func (scp *Scp) receiveFile(rd io.Reader, cw io.Writer, dstDir, dstName string, rcvFile *FileSet, outPipe io.Writer) (err error){
	//C command - file
	thisDstFile := ""
	if dstName == "" {
		thisDstFile = filepath.Join(dstDir, rcvFile.filename)
	}else{
		thisDstFile = filepath.Join(dstDir, dstName)
	}
	if scp.IsVerbose {
		scp.Println("Creating destination file: ", thisDstFile)
	}
	pb := NewProgressBarTo(rcvFile.filename, rcvFile.size, outPipe)
	if !scp.IsQuiet {
		pb.Update(0)
	}
	//TODO: mode here
	fw, err := os.Create(thisDstFile)
	if err != nil {
		return
	}
	defer fw.Close()
	reader := bufio.NewReader(rd)
	tot := int64(0)
	lastPercent := int64(0)
	var rb []byte
	for tot < rcvFile.size {
		rest := rcvFile.size - tot
		if rest < BUF_SIZE {
			rb = make([]byte, rest)
		}else{
			rb = make([]byte, BUF_SIZE)
		}
		n, err := reader.Read(rb)
		if err != nil {
			return err
		}
		wb := rb[:n]
		for i:=0;i < n;{
			wn, err := fw.Write(wb[i:])
			if err != nil {
				return err
			}
			i += wn
		}
		tot += int64(n)
		percent := (100 * tot) / rcvFile.size
		if percent > lastPercent {
			if !scp.IsQuiet {
				pb.Update(tot)
			}
		}
		lastPercent = percent
	}
	if scp.IsPreserve{
		if err := fw.Chmod(rcvFile.mode); err != nil {
			return err
		}
	}
	//close file writer & check error
	err = fw.Close()
	if err != nil {
		return
	}
	if scp.IsPreserve {
		if err := os.Chtimes(thisDstFile, rcvFile.atime, rcvFile.mtime); err != nil {
			return err
		}
	}
	if !scp.IsQuiet {
		pb.Update(tot)
		scp.Println() //new line
	}
	return
}
func (scp *Scp) parseCmdLine(rd io.Reader) (parts []string, err error){
	br := bufio.NewReader(rd)
	line, _, err := br.ReadLine()
	if err != nil {
		return
	}
	cmdLine := string(line)
	if scp.IsVerbose {
		scp.Printf("cmdFull:[%v]\n", cmdLine)
	}
	parts = strings.Split(string(cmdLine), " ")
	if cmdLine == "" || len(parts) == 0 {
		scp.Printf("Received OK \n")
		return
	}
	return
}
