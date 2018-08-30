package scp

import (
	"bufio"
	"errors"
	sshcon "github.com/takemxn/gssh/shared"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)
func (scp *Scp) openDstFromRemote(rd io.Reader, wd io.Writer) (err error) {
	dstFile := scp.dstFile
	errPipe := scp.Stderr
	outPipe := scp.Stdout
	dstFileInfo, err := os.Stat(dstFile)
	dstDir := dstFile
	var useSpecifiedFilename bool
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		} else {
			//OK - create file/dir
			useSpecifiedFilename = true
		}
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
	ce := make(chan error)
	scp.ce = ce
	go func() {
		cw := wd
		if err != nil {
			log.Println(err.Error())
			ce <- err
			return
		}
		r := rd
		if err != nil {
			log.Println("stdout err: "+err.Error()+" continue anyway")
			ce <- err
			return
		}
		if scp.IsVerbose {
			log.Println("Sending null byte")
		}
		err = sendByte(cw, 0)
		if err != nil {
			log.Println("Write error: "+err.Error())
			ce <- err
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
						log.Println("Received EOF from remote server")
					}
				} else {
					log.Println("Error reading standard input:", err)
					ce <- err
				}
				return
			}
			if n < 1 {
				log.Println("Error reading next byte from standard input")
				ce <- errors.New("Error reading next byte from standard input")
				return
			}
			cmd := cmdArr[0]
			if scp.IsVerbose {
				log.Printf("Sink: %s (%v)\n", string(cmd), cmd)
			}
			switch cmd {
			case 0x0:
				//continue
				if scp.IsVerbose {
					log.Printf("Received OK \n")
				}
			case 'E':
				//E command: go back out of dir
				dstDir = filepath.Dir(dstDir)
				if scp.IsVerbose {
					log.Printf("Received End-Dir\n")
				}
				err = sendByte(cw, 0)
				if err != nil {
					log.Println("Write error: %s", err.Error())
					ce <- err
					return
				}
			case 0xA:
				//0xA command: end?
				if scp.IsVerbose {
					log.Printf("Received All-done\n")
				}

				err = sendByte(cw, 0)
				if err != nil {
					log.Println("Write error: "+err.Error())
					ce <- err
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
							log.Println("Received EOF from remote server")
						}
					} else {
						log.Println("Error reading standard input:", err)
						ce <- err
					}
					return
				}
				//first line
				cmdFull := scanner.Text()
				if scp.IsVerbose {
					log.Printf("Details: %v\n", cmdFull)
				}
				//remainder, split by spaces
				parts := strings.SplitN(cmdFull, " ", 3)

				switch cmd {
				case 0x1:
					log.Printf("Received error message: %s\n", cmdFull[1:])
					ce <- errors.New(cmdFull[1:])
					return
				case 'D', 'C':
					mode, err := strconv.ParseInt(parts[0], 8, 32)
					if err != nil {
						log.Println("Format error: "+err.Error())
						ce <- err
						return
					}
					sizeUint, err := strconv.ParseUint(parts[1], 10, 64)
					size := int64(sizeUint)
					if err != nil {
						log.Println("Format error: "+err.Error())
						ce <- err
						return
					}
					rcvFilename := parts[2]
					if scp.IsVerbose {
						log.Printf("Mode: %d, size: %d, filename: %s\n", mode, size, rcvFilename)
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
						log.Println("Send error: "+err.Error())
						ce <- err
						return
					}
					if cmd == 'C' {
						//C command - file
						thisDstFile := filepath.Join(dstDir, filename)
						if scp.IsVerbose {
							log.Println("Creating destination file: ", thisDstFile)
						}
						tot := int64(0)
						pb := NewProgressBarTo(filename, size, outPipe)
						pb.Update(0)

						//TODO: mode here
						fw, err := os.Create(thisDstFile)
						if err != nil {
							ce <- err
							log.Println("File creation error: "+err.Error())
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
								log.Println("Read error: "+err.Error())
								ce <- err
								return
							}
							tot += int64(n)
							//write to file
							_, err = fw.Write(b[:n])
							if err != nil {
								log.Println("Write error: "+err.Error())
								ce <- err
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
							log.Println(err.Error())
							ce <- err
							return
						}
						//get next byte from channel reader
						nb := make([]byte, 1)
						_, err = r.Read(nb)
						if err != nil {
							log.Println(err.Error())
							ce <- err
							return
						}
						//TODO check value received in nb
						//send null-byte back
						_, err = cw.Write([]byte{0})
						if err != nil {
							log.Println("Send null-byte error: "+err.Error())
							ce <- err
							return
						}
						pb.Update(tot)
						log.Println(errPipe) //new line
					} else {
						//D command (directory)
						thisDstFile := filepath.Join(dstDir, filename)
						fileMode := os.FileMode(uint32(mode))
						err = os.MkdirAll(thisDstFile, fileMode)
						if err != nil {
							log.Println("Mkdir error: "+err.Error())
							ce <- err
							return
						}
						dstDir = thisDstFile
					}
				default:
					log.Printf("Command '%v' NOT implemented\n", cmd)
					return
				}
			}
			first = false
		}
	}()
	return
}
func (scp *Scp) openDstToRemote() (r io.Reader, w io.WriteCloser, err error) {
	conn, err := sshcon.Connect2(scp.dstUser, scp.dstHost, scp.Port)
	if err != nil {
		log.Printf("unable to create session: %s", err)
		return nil, nil, err
	}
	s, err := conn.NewSession()
	if err != nil {
		return nil, nil, err
	} else if scp.IsVerbose {
		log.Println(scp.Stderr, "Got session")
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
	err = s.Start("/usr/bin/scp " + remoteOpts + " " + scp.dstFile)
	if err != nil {
		log.Println(scp.Stderr, "Failed to run remote scp: "+err.Error())
	}
	scp.ses = s
	return
}
func (scp *Scp) OpenDst() (rw *ReadWriter, err error) {
	if scp.dstHost != "" {
		r, w, err := scp.openDstToRemote()
		if err != nil {
			return  nil, err
		}
		rw = NewReadWriter(r, w)
	} else {
		r, w := io.Pipe()
		r2, w2 := io.Pipe()
		err := scp.openDstFromRemote(r, w2)
		if err != nil {
			return nil, err
		}
		rw = NewReadWriter(r2, w)
	}
	return
}
