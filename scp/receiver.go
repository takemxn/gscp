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
func (scp *Scp) openLocalReceiver(rd *Channel, cw *Channel, rCh chan error) (err error) {
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
	go func() {
		defer func(){
			rCh <- nil
		}()
		if scp.IsVerbose {
			scp.Println("Sending null byte")
		}
		err = sendByte(cw, 0)
		if err != nil {
			rCh <- err
			return
		}
		fs := new(FileSet)
		for {
			b := make([]byte, 1)
			n, err := rd.Read(b)
			if err != nil {
				rCh <- err
				return
			}
			if n == 0 {
				return
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
					rCh <- err
					return
				}
				scp.Println(string(line))
			case 'E':
				//E command: go back out of dir
				dstDir = filepath.Dir(dstDir)
				if scp.IsVerbose {
					scp.Printf("Received End-Dir\n")
				}
				err = sendByte(cw, 0)
				if err != nil {
					scp.Println("Write error: %s", err.Error())
					rCh <- err
					return
				}
			case 0xA:
				//0xA command: end?
				if scp.IsVerbose {
					scp.Printf("Received All-done\n")
				}
				err = sendByte(cw, 0)
				if err != nil {
					rCh <- err
					return
				}
				return
			case 'D':
				parts, err := scp.parseCmdLine(rd)
				if err != nil {
					rCh <- err
					return
				}
				fs.mode, fs.size, fs.filename, err = scp.parseCmd(parts)
				if err != nil {
					rCh <- err
					return
				}
				if !scp.IsRecursive {
					rCh <- fmt.Errorf("%q/%q is not aregular file", dstDir, fs.filename)
					return
				}
				fileMode := os.FileMode(uint32(fs.mode))
				if dstFileNotExist {
					if scp.IsVerbose {
						scp.Printf("makdir %q\n", dstDir)
					}
					err = os.Mkdir(dstDir, fileMode)
					if err != nil {
						rCh <- err
						return
					}
				}else if !dstFileInfo.IsDir(){
					rCh <- fmt.Errorf("%q: Not a directory", dstFile)
					return
				}
				//D command (directory)
				thisDstFile := filepath.Join(dstDir, fs.filename)
				err = os.MkdirAll(thisDstFile, fileMode)
				if err != nil {
					rCh <- err
					return
				}
				if scp.IsPreserve {
					if err := os.Chtimes(thisDstFile, fs.atime, fs.mtime); err != nil {
						rCh <- err
						return
					}
				}
				dstDir = thisDstFile
				err = sendByte(cw, 0)
				if err != nil {
					rCh <- err
					return
				}
			case 'C':
				parts, err := scp.parseCmdLine(rd)
				if err != nil {
					rCh <- err
					return
				}
				fs.mode, fs.size, fs.filename, err = scp.parseCmd(parts)
				if err != nil {
					rCh <- err
					return
				}
				err = sendByte(cw, 0)
				if err != nil {
					scp.Println("Write error: %s", err.Error())
					rCh <- err
					return
				}
				err = scp.receiveFile(rd, cw, dstDir, dstName, fs, errPipe)
				if err != nil {
					rCh <- err
					return
				}
			case 'T':
				parts, err := scp.parseCmdLine(rd)
				if err != nil {
					rCh <- err
					return
				}
				// modification time
				t, err := strconv.ParseUint(parts[0], 10, 64)
				if err != nil {
					rCh <- err
					return
				}
				fs.mtime = time.Unix(int64(t), 0)
				// access time
				t, err = strconv.ParseUint(parts[2], 10, 64)
				if err != nil {
					rCh <- err
					return
				}
				fs.atime = time.Unix(int64(t), 0)
				err = sendByte(cw, 0)
				if err != nil {
					scp.Println("Write error: %s", err.Error())
					rCh <- err
					return
				}
			default :
				rCh <- fmt.Errorf("Command '%x' NOT implemented\n", cmd)
				return
			}
		}
	}()
	return
}
func (scp *Scp) openRemoteReceiver(in, out *Channel, rCh chan error) (err error) {
	password := scp.Password
	if password == "" {
		password = scp.config.GetPassword(scp.dstUser,  scp.dstHost, scp.Port)
	}
	ci := com.NewConnectInfo(scp.dstUser, scp.dstHost, scp.Port, password)
	conn, err := ci.Connect()
	if err != nil {
		scp.Printf("unable to create session: %s", err)
		return err
	}
	s, err := conn.NewSession()
	if err != nil {
		return err
	} else if scp.IsVerbose {
		scp.Println("Got receiver session")
	}
	w, err := s.StdinPipe()
	if err != nil {
		return err
	}
	r, err := s.StdoutPipe()
	if err != nil {
		return err
	}
	go func(){
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
				scp.Println("scp write error", err)
				return
			}
		}
	}()
	go func(){
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
	remoteOpts := "-pt"
	if scp.IsQuiet {
		remoteOpts += "q"
	}
	if scp.IsRecursive {
		remoteOpts += "r"
	}
	go func(){
		err = s.Run("/usr/bin/scp " + remoteOpts + " " + scp.dstFile)
		if err != nil {
			rCh <- err
		}
	}()
	return
}
func (scp *Scp) openReceiver(rCh chan error) (in *Channel, out *Channel, err error) {
	in = NewChannel()
	out = NewChannel()
	if scp.dstHost != "" {
		err = scp.openRemoteReceiver(in, out, rCh)
		if err != nil {
			return nil, nil, err
		}
	} else {
		err = scp.openLocalReceiver(in, out, rCh)
		if err != nil {
			return
		}
	}
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
		scp.Printf("Mode: %d, size: %d, filename: %s\n", mode, size, filename)
	}
	return
}
func (scp *Scp) receiveFile(rd io.Reader, cw io.Writer, dstDir, dstName string, rcvFile *FileSet, outPipe io.Writer) (err error){
	//C command - file
	thisDstFile := ""
	if dstName == "" {
		thisDstFile = filepath.Join(dstDir, rcvFile.filename)
	fmt.Println("thisDstFile1:", thisDstFile)
	}else{
		thisDstFile = filepath.Join(dstDir, dstName)
	fmt.Println("thisDstFile2:", thisDstFile)
	}
	if scp.IsVerbose {
		scp.Println("Creating destination file: ", thisDstFile)
	}
	pb := NewProgressBarTo(rcvFile.filename, rcvFile.size, outPipe)
	pb.Update(0)

	//TODO: mode here
	fw, err := os.Create(thisDstFile)
	if err != nil {
		return
	}
	defer fw.Close()

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
		n, err := rd.Read(rb)
		if err != nil {
			return err
		}
		_, err = fw.Write(rb[:n])
		if err != nil {
			return err
		}
		tot += int64(n)
		percent := (100 * tot) / rcvFile.size
		if percent > lastPercent {
			pb.Update(tot)
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
	_, err = cw.Write([]byte{0})
	if err != nil {
		return
	}
	//get next byte from channel reader
	b := make([]byte, 1)
	_, err = rd.Read(b)
	if err != nil {
		return
	}
	if scp.IsPreserve {
		if err := os.Chtimes(thisDstFile, rcvFile.atime, rcvFile.mtime); err != nil {
			return err
		}
	}
	pb.Update(tot)
	scp.Println() //new line
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
