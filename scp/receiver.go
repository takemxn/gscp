package scp

import (
	"fmt"
	"bufio"
	"errors"
	com "github.com/takemxn/gssh/shared"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)
func (scp *Scp) openLocalReceiver(rd io.Reader, wd io.Writer, rs chan error) (err error) {
	dstFile := scp.dstFile
	errPipe := scp.Stderr
	outPipe := scp.Stdout
	dstFileInfo, e := os.Stat(dstFile)
	dstDir := dstFile
	var useSpecifiedFilename bool
	if e != nil {
		if !os.IsNotExist(e) {
			return e
		}
		//OK - create file/dir
		useSpecifiedFilename = true
	} else if dstFileInfo.IsDir() {
		//ok - use name of srcFile
		//dstFile = filepath.Join(dstFile, filepath.Base(srcFile))
		dstDir = dstFile
		//MUST use received filename instead
		//TODO should this be from USR?
		useSpecifiedFilename = false
	} else {
		dstDir = filepath.Dir(dstFile)
		useSpecifiedFilename = true
	}
	go func() {
		defer func(){
			sendByte(wd, 0)
			close(rs)
		}()

		cw := wd
		r := rd
		if scp.IsVerbose {
			fmt.Println("Sending null byte")
		}
		err = sendByte(cw, 0)
		if err != nil {
			fmt.Println("Write error: "+err.Error())
			rs <- err
			return
		}
		//defer r.Close()
		//use a scanner for processing individual commands, but not files themselves
		scanner := bufio.NewScanner(r)
		more := true
		first := true
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
					fmt.Println("Error reading standard input:", err)
					rs <- err
				}
				return
			}
			if n < 1 {
				fmt.Println("Error reading next byte from standard input")
				rs <- errors.New("Error reading next byte from standard input")
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
					rs <- err
					return
				}
			case 0xA:
				//0xA command: end?
				if scp.IsVerbose {
					fmt.Printf("Received All-done\n")
				}

				err = sendByte(cw, 0)
				if err != nil {
					fmt.Println("Write error: "+err.Error())
					rs <- err
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
						fmt.Println("Error reading standard input:", err)
						rs <- err
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
					fmt.Printf("Received error message: %s\n", cmdFull[1:])
					rs <- errors.New(cmdFull[1:])
					return
				case 'D', 'C':
					mode, err := strconv.ParseInt(parts[0], 8, 32)
					if err != nil {
						fmt.Println("Format error: "+err.Error())
						rs <- err
						return
					}
					sizeUint, err := strconv.ParseUint(parts[1], 10, 64)
					size := int64(sizeUint)
					if err != nil {
						fmt.Println("Format error: "+err.Error())
						rs <- err
						return
					}
					rcvFilename := parts[2]
					if scp.IsVerbose {
						fmt.Printf("Mode: %d, size: %d, filename: %s\n", mode, size, rcvFilename)
					}
					var filename string
					//use the specified filename from the destination (only for top-level item)
					if useSpecifiedFilename && first {
						filename = filepath.Base(dstFile)
					} else {
						filename = rcvFilename
					}
					err = sendByte(cw, 0)
					if err != nil {
						fmt.Println("Send error: "+err.Error())
						rs <- err
						return
					}
					if cmd == 'C' {
						//C command - file
						thisDstFile := filepath.Join(dstDir, filename)
						if scp.IsVerbose {
							fmt.Fprintln(scp.Stderr, "Creating destination file: ", thisDstFile)
						}
						tot := int64(0)
						pb := NewProgressBarTo(filename, size, outPipe)
						pb.Update(0)

						//TODO: mode here
						fw, err := os.Create(thisDstFile)
						if err != nil {
							fmt.Fprintln(scp.Stderr, "File creation error: ",err)
							rs <- err
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
								fmt.Fprintln(scp.Stderr, "Read error: ",err)
								rs <- err
								return
							}
							tot += int64(n)
							//write to file
							_, err = fw.Write(b[:n])
							if err != nil {
								fmt.Println("Write error: "+err.Error())
								rs <- err
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
							fmt.Println(err.Error())
							rs <- err
							return
						}
						//get next byte from channel reader
						nb := make([]byte, 1)
						_, err = r.Read(nb)
						if err != nil {
							fmt.Println(err.Error())
							rs <- err
							return
						}
						//TODO check value received in nb
						//send null-byte back
						_, err = cw.Write([]byte{0})
						if err != nil {
							fmt.Println("Send null-byte error: "+err.Error())
							rs <- err
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
							fmt.Println("Mkdir error: "+err.Error())
							rs <- err
							return
						}
						dstDir = thisDstFile
					}
				default:
					fmt.Printf("Command '%v' NOT implemented\n", cmd)
					return
				}
			}
			first = false
		}
	}()
	return
}
func (scp *Scp) openRemoteReceiver(rs chan error) (r io.Reader, w io.WriteCloser, err error) {
	conn, err := com.Connect2(scp.dstUser, scp.dstHost, scp.Port)
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
			fmt.Fprintln(scp.Stderr, "Failed to run remote scp: "+err.Error())
			rs <- err
		}
		rs <- s.Wait()
	}()
	return
}
func (scp *Scp) openReceiver(rs chan error) (rw *ReadWriter, err error) {
	if scp.dstHost != "" {
		r, w, err := scp.openRemoteReceiver(rs)
		if err != nil {
			return  nil, err
		}
		rw = NewReadWriter(r, w)
	} else {
		r, w := io.Pipe()
		r2, w2 := io.Pipe()
		err := scp.openLocalReceiver(r, w2, rs)
		if err != nil {
			return nil, err
		}
		rw = NewReadWriter(r2, w)
	}
	return
}
