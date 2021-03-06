package scp

import (
	"fmt"
	"io"
	"os"
	"time"
)


const (
	DEFAULT_FORMAT = "\r%s   % 3d %%  %.2f kb %0.2f kb/s %v      "
)

type ProgressBar struct {
	Out       io.Writer
	Format    string
	Subject   string
	StartTime time.Time
	Size      int64
}

func NewProgressBarTo(subject string, size int64, outPipe io.Writer) ProgressBar {
	return ProgressBar{outPipe, DEFAULT_FORMAT, subject, time.Now(), size}
}

func NewProgressBar(subject string, size int64) ProgressBar {
	return NewProgressBarTo(subject, size, os.Stdout)
}

func (pb ProgressBar) Update(tot int64) {
	percent := int64(0)
	if pb.Size > int64(0) {
		percent = (int64(100) * tot) / pb.Size
	}
	totTime := time.Now().Sub(pb.StartTime)
	totK := float64(tot / 1024)
	spd := totK / totTime.Seconds()
	fmt.Fprintf(pb.Out, pb.Format, pb.Subject, percent, totK, spd, totTime)
}
