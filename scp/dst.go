package scp

import (
	"bufio"
	"errors"
	"fmt"
	sshcon "github.com/takemxn/gssh/shared"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (scp *Scp) openDstFromRemote(reader io.Reader, writer io.Writer) (err error) {
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
	go func() {
		cw := writer
		if err != nil {
			fmt.Fprintln(errPipe, err.Error())
			ce <- err
			return
		}
		r := reader
		if err != nil {
			fmt.Fprintln(errPipe, "stdout err: "+err.Error()+" continue anyway")
			ce <- err
			return
		}
		if scp.IsVerbose {
			fmt.Fprintln(errPipe, "Sending null byte")
		}
		err = sendByte(cw, 0)
		if err != nil {
			fmt.Fprintln(errPipe, "Write error: "+err.Error())
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
						fmt.Fprintln(errPipe, "Received EOF from remote server")
					}
				} else {
					fmt.Fprintln(errPipe, "Error reading standard input:", err)
					ce <- err
				}
				return
			}
			if n < 1 {
				fmt.Fprintln(errPipe, "Error reading next byte from standard input")
				ce <- errors.New("Error reading next byte from standard input")
				return
			}
			cmd := cmdArr[0]
			if scp.IsVerbose {
				fmt.Fprintf(errPipe, "Sink: %s (%v)\n", string(cmd), cmd)
			}
			switch cmd {
			case 0x0:
				//continue
				if scp.IsVerbose {
					fmt.Fprintf(errPipe, "Received OK \n")
				}
			case 'E':
				//E command: go back out of dir
				dstDir = filepath.Dir(dstDir)
				if scp.IsVerbose {
					fmt.Fprintf(errPipe, "Received End-Dir\n")
				}
				err = sendByte(cw, 0)
				if err != nil {
					fmt.Fprintln(errPipe, "Write error: %s", err.Error())
					ce <- err
					return
				}
			case 0xA:
				//0xA command: end?
				if scp.IsVerbose {
					fmt.Fprintf(errPipe, "Received All-done\n")
				}

				err = sendByte(cw, 0)
				if err != nil {
					fmt.Fprintln(errPipe, "Write error: "+err.Error())
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
							fmt.Fprintln(errPipe, "Received EOF from remote server")
						}
					} else {
						fmt.Fprintln(errPipe, "Error reading standard input:", err)
						ce <- err
					}
					return
				}
				//first line
				cmdFull := scanner.Text()
				if scp.IsVerbose {
					fmt.Fprintf(errPipe, "Details: %v\n", cmdFull)
				}
				//remainder, split by spaces
				parts := strings.SplitN(cmdFull, " ", 3)

				switch cmd {
				case 0x1:
					fmt.Fprintf(errPipe, "Received error message: %s\n", cmdFull[1:])
					ce <- errors.New(cmdFull[1:])
					return
				case 'D', 'C':
					mode, err := strconv.ParseInt(parts[0], 8, 32)
					if err != nil {
						fmt.Fprintln(errPipe, "Format error: "+err.Error())
						ce <- err
						return
					}
					sizeUint, err := strconv.ParseUint(parts[1], 10, 64)
					size := int64(sizeUint)
					if err != nil {
						fmt.Fprintln(errPipe, "Format error: "+err.Error())
						ce <- err
						return
					}
					rcvFilename := parts[2]
					if scp.IsVerbose {
						fmt.Fprintf(errPipe, "Mode: %d, size: %d, filename: %s\n", mode, size, rcvFilename)
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
						fmt.Fprintln(errPipe, "Send error: "+err.Error())
						ce <- err
						return
					}
					if cmd == 'C' {
						//C command - file
						thisDstFile := filepath.Join(dstDir, filename)
						if scp.IsVerbose {
							fmt.Fprintln(errPipe, "Creating destination file: ", thisDstFile)
						}
						tot := int64(0)
						pb := NewProgressBarTo(filename, size, outPipe)
						pb.Update(0)

						//TODO: mode here
						fw, err := os.Create(thisDstFile)
						if err != nil {
							ce <- err
							fmt.Fprintln(errPipe, "File creation error: "+err.Error())
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
								fmt.Fprintln(errPipe, "Read error: "+err.Error())
								ce <- err
								return
							}
							tot += int64(n)
							//write to file
							_, err = fw.Write(b[:n])
							if err != nil {
								fmt.Fprintln(errPipe, "Write error: "+err.Error())
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
							fmt.Fprintln(errPipe, err.Error())
							ce <- err
							return
						}
						//get next byte from channel reader
						nb := make([]byte, 1)
						_, err = r.Read(nb)
						if err != nil {
							fmt.Fprintln(errPipe, err.Error())
							ce <- err
							return
						}
						//TODO check value received in nb
						//send null-byte back
						_, err = cw.Write([]byte{0})
						if err != nil {
							fmt.Fprintln(errPipe, "Send null-byte error: "+err.Error())
							ce <- err
							return
						}
						pb.Update(tot)
						fmt.Fprintln(errPipe) //new line
					} else {
						//D command (directory)
						thisDstFile := filepath.Join(dstDir, filename)
						fileMode := os.FileMode(uint32(mode))
						err = os.MkdirAll(thisDstFile, fileMode)
						if err != nil {
							fmt.Fprintln(errPipe, "Mkdir error: "+err.Error())
							ce <- err
							return
						}
						dstDir = thisDstFile
					}
				default:
					fmt.Fprintf(errPipe, "Command '%v' NOT implemented\n", cmd)
					return
				}
			}
			first = false
		}
	}()
	return
}
func (scp *Scp) openDstToRemote(r io.Reader, w io.Writer) (err error) {
	conn, err := sshcon.Connect2(scp.dstUser, scp.dstHost, scp.Port)
	if err != nil {
		log.Printf("unable to create session: %s", err)
		return err
	}
	s, err := conn.NewSession()
	if err != nil {
		return err
	} else if scp.IsVerbose {
		fmt.Fprintln(scp.Stderr, "Got session")
	}
	in, err := s.StdinPipe()
	if err != nil {
		return err
	}
	out, err := s.StdoutPipe()
	if err != nil {
		return err
	}
	go io.Copy(in, r)
	go io.Copy(w, out)
	remoteOpts := "-t"
	if scp.IsQuiet {
		remoteOpts += "q"
	}
	if scp.IsRecursive {
		remoteOpts += "r"
	}
	err = s.Start("/usr/bin/scp " + remoteOpts + " " + scp.dstFile)
	if err != nil {
		fmt.Fprintln(scp.Stderr, "Failed to run remote scp: "+err.Error())
	}
	return
}
func (scp *Scp) OpenDst() (r io.ReadCloser, w io.WriteCloser, err error) {
	r, w = io.Pipe()
	if scp.dstHost != "" {
		err = scp.openDstToRemote(r, w)
		if err != nil {
			return nil, nil, err
		}
	} else {
		err := scp.openDstFromRemote(r, w)
		if err != nil {
			return nil, nil, err
		}
	}
	return
}
