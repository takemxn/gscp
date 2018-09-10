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
	var dstFileNotExist bool
	if e != nil {
		if !os.IsNotExist(e) {
			return e
		}
		//OK - create file/dir
		dstFileNotExist = true
		dstDir = filepath.Dir(dstFile)
	} else if dstFileInfo.IsDir() {
		//ok - use name of srcFile
		//dstFile = filepath.Join(dstFile, filepath.Base(srcFile))
		dstDir = dstFile
		//MUST use received filename instead
		//TODO should this be from USR?
	} else if dstFileInfo.Mode().IsRegular() {
		dstDir = filepath.Dir(dstFile)
	}else{
		return errors.New("spcified file was not dir or regular file!!")
	}
	go func() {
		defer func(){
			close(rCh)
		}()
		if scp.IsVerbose {
			scp.Println("Sending null byte")
		}
		err = sendByte(cw, 0)
		if err != nil {
			rCh <- err
			return
		}
		//defer r.Close()
		//use a scanner for processing individual commands, but not files themselves
		fs := new(FileSet)
		first := false
		//scanner := bufio.NewScanner(rd)
		//for scanner.Scan() {
		//	cmdFull := scanner.Text()
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
				scp.Printf("cmd : [%v]\n", string(cmd))
			}
			switch cmd {
			case 0x0:
				//continue
				if scp.IsVerbose {
					scp.Printf("Received OK \n")
				}
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
				if !scp.IsRecursive && first {
					rCh <- fmt.Errorf("%q/%q is not aregular file", dstDir, fs.filename)
					return
				}
				fileMode := os.FileMode(uint32(fs.mode))
				if first {
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
						rCh <- fmt.Errorf("%q: Not a directory", dstDir)
						return
					}
				}
				first = false
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
				err = scp.receiveFile(rd, cw, dstDir, fs, errPipe)
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
	go io.Copy(in, r)
	go io.Copy(w, out)
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
		rCh <- nil
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
func (scp *Scp) receiveFile(rd io.Reader, cw io.Writer, dstDir string, fs *FileSet, outPipe io.Writer) (err error){
	//C command - file
	thisDstFile := filepath.Join(dstDir, fs.filename)
	if scp.IsVerbose {
		scp.Println("Creating destination file: ", thisDstFile)
	}
	pb := NewProgressBarTo(fs.filename, fs.size, outPipe)
	pb.Update(0)

	//TODO: mode here
	fw, err := os.Create(thisDstFile)
	if err != nil {
		return
	}
	defer fw.Close()

	//buffered by 4096 bytes
	tot := int64(0)
	lastPercent := int64(0)
	var rb []byte
	for tot < fs.size {
		rest := fs.size - tot
		if rest < 4096 {
			rb = make([]byte, rest)
		}else{
			rb = make([]byte, 4096)
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
		percent := (100 * tot) / fs.size
		if percent > lastPercent {
			pb.Update(tot)
		}
		lastPercent = percent
	}
	if scp.IsPreserve{
		if err := fw.Chmod(fs.mode); err != nil {
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
		if err := os.Chtimes(thisDstFile, fs.atime, fs.mtime); err != nil {
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
