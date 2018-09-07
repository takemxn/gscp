package scp

import (
	"fmt"
	"bufio"
	"errors"
	com "github.com/takemxn/gssh/common"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)
func (scp *Scp) openLocalReceiver(rd io.Reader, cw io.Writer, rCh chan error) (err error) {
	dstFile := scp.dstFile
	errPipe := scp.Stderr
	outPipe := scp.Stdout
	dstFileInfo, e := os.Stat(dstFile)
	dstDir := dstFile
	var useSpecifiedFilename bool
	var dstFileNotExist bool
	if e != nil {
		if !os.IsNotExist(e) {
			return e
		}
		//OK - create file/dir
		useSpecifiedFilename = true
		dstFileNotExist = true
		dstFile = ""
	} else if dstFileInfo.IsDir() {
		//ok - use name of srcFile
		//dstFile = filepath.Join(dstFile, filepath.Base(srcFile))
		dstDir = dstFile
		//MUST use received filename instead
		//TODO should this be from USR?
		useSpecifiedFilename = false
	} else if dstFileInfo.Mode().IsRegular() {
		dstDir = filepath.Dir(dstFile)
		useSpecifiedFilename = true
	}else{
		return errors.New("spcified file was not dir or regular file!!")
	}
	go func() {
		defer func(){
			close(rCh)
		}()
		r := rd
		if scp.IsVerbose {
			fmt.Println("Sending null byte")
		}
		err = sendByte(cw, 0)
		if err != nil {
			rCh <- err
			return
		}
		//defer r.Close()
		//use a scanner for processing individual commands, but not files themselves
		scanner := bufio.NewScanner(r)
		first := false
		var atime, mtime uint64
		for scanner.Scan() {
			cmdFull := scanner.Text()
			parts := strings.Split(cmdFull, " ")
			if len(parts) == 0 {
				fmt.Printf("Received OK \n")
				continue
			}
			atime = 0
			mtime = 0
			cmd := parts[0][0:1]
			switch cmd {
			case 0x0:
				//continue
				if scp.IsVerbose {
					fmt.Printf("Received OK \n")
				}
			case 'E':
				//E command: go back out of dir
				dstDir = filepath.Dir(dstDir)
				if scp.IsVerbose {
					fmt.Printf("Received End-Dir\n")
				}
				err = sendByte(cw, 0)
				if err != nil {
					fmt.Println("Write error: %s", err.Error())
					rCh <- err
					return
				}
			case 0xA:
				//0xA command: end?
				if scp.IsVerbose {
					fmt.Printf("Received All-done\n")
				}
				err = sendByte(cw, 0)
				if err != nil {
					rCh <- err
					return
				}
				return
			case 'D':
				mode, size, rcvDirname, err := parseCmd(parts)
				if err != nil {
					rCh <- err
					return
				}
			case 'C':
				mode, size, rcvFilename, err := parseCmd(parts)
				if err != nil {
					rCh <- err
					return
				}
			case 'T':
				var t uint
				t, err = strconv.ParseUint(parts[0][0:1], 10, 64)
				if err != nil {
					rCh <- err
					return
				}
				atime := int64(t)
				t, err = strconv.ParseUint(parts[2], 10, 64)
				if err != nil {
					rCh <- err
					return
				}
				mtime := int64(t)
			default :
				rCh <- fmt.Errorf("Command '%v' NOT implemented\n", cmd)
				return
			}
		}
		for more {
			cmdArr := make([]byte, 1)
			n, err := r.Read(cmdArr)
			if err != nil {
				if err == io.EOF {
					//no problem.
					if scp.IsVerbose {
						fmt.Println("Received EOF from remote server")
					}
				} else {
					rCh <- err
				}
				return
			}
			if n < 1 {
				rCh <- errors.New("Error reading next byte from standard input")
				return
			}
			cmd := cmdArr[0]
			if scp.IsVerbose {
				fmt.Printf("Sink: %s (%v)\n", string(cmd), cmd)
			}
			switch cmd {
			case 0x0:
				//continue
				if scp.IsVerbose {
					fmt.Printf("Received OK \n")
				}
			case 'E':
				//E command: go back out of dir
				dstDir = filepath.Dir(dstDir)
				if scp.IsVerbose {
					fmt.Printf("Received End-Dir\n")
				}
				err = sendByte(cw, 0)
				if err != nil {
					fmt.Println("Write error: %s", err.Error())
					rCh <- err
					return
				}
			case 0xA:
				//0xA command: end?
				if scp.IsVerbose {
					fmt.Printf("Received All-done\n")
				}

				err = sendByte(cw, 0)
				if err != nil {
					rCh <- err
					return
				}

				return
			default:
				scanner.Scan()
				err = scanner.Err()
				if err != nil {
					if err == io.EOF {
						//no problem.
						if scp.IsVerbose {
							fmt.Println("Received EOF from remote server")
						}
					} else {
						rCh <- err
					}
					return
				}
				//first line
				cmdFull := scanner.Text()
				if scp.IsVerbose {
					fmt.Printf("Details: %v\n", cmdFull)
				}
				//remainder, split by spaces
				parts := strings.SplitN(cmdFull, " ", 3)

				switch cmd {
				case 0x1:
					rCh <- errors.New(cmdFull[1:])
					return
				case 'D', 'C':
					mode, err := strconv.ParseInt(parts[0], 8, 32)
					if err != nil {
						rCh <- err
						return
					}
					sizeUint, err := strconv.ParseUint(parts[1], 10, 64)
					size := int64(sizeUint)
					if err != nil {
						rCh <- err
						return
					}
					rcvFilename := parts[2]
					if scp.IsVerbose {
						fmt.Printf("Mode: %d, size: %d, filename: %s\n", mode, size, rcvFilename)
					}
					var filename string
					//use the specified filename from the destination (only for top-level item)
					if dstFileNotExist && scp.IsRecursive && first{
						err = os.Mkdir(dstFile, fileMode)
						if err != nil {
							rCh <- err
							return
						}
					}
					if useSpecifiedFilename {
						if dstFileNotExist {
							filename = dstFile
						}else{
							filename = filepath.Base(dstFile)
						}
					} else {
						filename = rcvFilename
					}
					err = sendByte(cw, 0)
					if err != nil {
						rCh <- err
						return
					}
					if cmd == 'C' {
						//C command - file
						thisDstFile := filepath.Join(dstDir, filename)
						if scp.IsVerbose {
							fmt.Fprintln(scp.Stderr, "Creating destination file: ", thisDstFile)
						}
						tot := int64(0)
						pb := NewProgressBarTo(rcvFilename, size, outPipe)
						pb.Update(0)

						//TODO: mode here
						fw, err := os.Create(thisDstFile)
						if err != nil {
							rCh <- err
							return
						}
						defer fw.Close()

						//buffered by 4096 bytes
						bufferSize := int64(4096)
						lastPercent := int64(0)
						for tot < size {
							if bufferSize > size-tot {
								bufferSize = size - tot
							}
							b := make([]byte, bufferSize)
							n, err = r.Read(b)
							if err != nil {
								rCh <- err
								return
							}
							tot += int64(n)
							//write to file
							_, err = fw.Write(b[:n])
							if err != nil {
								rCh <- err
								return
							}
							percent := (100 * tot) / size
							if percent > lastPercent {
								pb.Update(tot)
							}
							lastPercent = percent
						}
						//close file writer & check error
						err = fw.Close()
						if err != nil {
							rCh <- err
							return
						}
						//get next byte from channel reader
						nb := make([]byte, 1)
						_, err = r.Read(nb)
						if err != nil {
							rCh <- err
							return
						}
						//TODO check value received in nb
						//send null-byte back
						_, err = cw.Write([]byte{0})
						if err != nil {
							rCh <- err
							return
						}
						pb.Update(tot)
						fmt.Println(errPipe) //new line
					} else {
						//D command (directory)
						thisDstFile := filepath.Join(dstDir, filename)
						fileMode := os.FileMode(uint32(mode))
						err = os.MkdirAll(thisDstFile, fileMode)
						if err != nil {
							rCh <- err
							return
						}
						dstDir = thisDstFile
					}
				default:
					rCh <- fmt.Errorf("Command '%v' NOT implemented\n", cmd)
					return
				}
			}
		}
	}()
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
		fmt.Printf("unable to create session: %s", err)
		return nil, nil, err
	}
	s, err := conn.NewSession()
	if err != nil {
		return nil, nil, err
	} else if scp.IsVerbose {
		fmt.Fprintln(scp.Stderr, "Got receiver session")
	}
	w, err = s.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	r, err = s.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	remoteOpts := "-t"
	if scp.IsQuiet {
		remoteOpts += "q"
	}
	if scp.IsRecursive {
		remoteOpts += "r"
	}
	go func(){
		err = s.Start("/usr/bin/scp " + remoteOpts + " " + scp.dstFile)
		if err != nil {
			rCh <- err
		}
		rCh <- s.Wait()
	}()
	return
}
func (scp *Scp) openReceiver(rCh chan error) (rw *ReadWriter, err error) {
	if scp.dstHost != "" {
		r, w, err := scp.openRemoteReceiver(rCh)
		if err != nil {
			return  nil, err
		}
		rw = NewReadWriter(r, w)
	} else {
		r, w := io.Pipe()
		r2, w2 := io.Pipe()
		err := scp.openLocalReceiver(r, w2, rCh)
		if err != nil {
			return nil, err
		}
		rw = NewReadWriter(r2, w)
	}
	return
}
func parseCmd(cmdStr []string) (mode int, size int64, filename string, err error){
	mode, err := strconv.ParseInt(cmdStr[0][1:0], 8, 32)
	if err != nil {
		return
	}
	sizeUint, err := strconv.ParseUint(cmdStr[1], 10, 64)
	size = int64(sizeUint)
	if err != nil {
		return
	}
	filename := cmdStr[2]
	if scp.IsVerbose {
		fmt.Printf("Mode: %d, size: %d, filename: %s\n", mode, size, filename)
	}
}
