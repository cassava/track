// Copyright (c) 2013, Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

// Track is a program to track the time you spend on a project by storing
// the start and end times in a CSV file.
//
// The aim of track is to make it easy to manage your time.
package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"time"
)

type FormatError struct {
	BadLines  []int
	LastIsBad bool
}

func (e *FormatError) JustIncomplete() bool {
	return len(e.BadLines) == 1 && e.LastIsBad
}

func (e *FormatError) Error() string {
	if e.JustIncomplete() {
		return "last entry is incomplete"
	} else {
		if len(e.BadLines) == 1 {
			return fmt.Sprint("incomplete or invalid entry on line ", e.BadLines[0])
		} else {
			return fmt.Sprint("incomplete or invalid entries on lines ", spokenList(e.BadLines))
		}
	}
}

var which = map[string]func() error{
	"begin":  Begin,
	"end":    End,
	"fork":   Fork,
	"list":   List,
	"next":   Next,
	"run":    Run,
	"status": Status,
	"total":  Total,
	"verify": Verify,
	"wait":   Wait,
}

const timeFormat = "2006-01-02 15:04:05 MST"

// Configuration variables which are read from the command line.
var (
	helpFlag  = false
	quietFlag = false
	failFlag  = false
	pathArg   = "TIMES.csv"
)

func init() {
	flag.Usage = Help
	flag.BoolVar(&failFlag, "fail", failFlag, "fail if there is any error in the times file")
	flag.BoolVar(&helpFlag, "help", helpFlag, "print this usage text for track")
	flag.BoolVar(&quietFlag, "quiet", quietFlag, "do not print informative messages")
}

func main() {
	var command = Status

	flag.Parse()
	if helpFlag {
		Help()
		return
	}

	args := flag.Args()
	n := flag.NArg()
	if n > 0 {
		command = which[args[0]]
		if command == nil || n > 3 {
			Help()
			os.Exit(2)
		}

		if n == 3 {
			pathArg = args[1]
		}
	}

	err := command()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func Help() {
	fmt.Println(`Usage: track [command [file]]

The default command is:
	track status TIMES.csv

Commands available are:
    begin   begin a new time entry
    end     complete the begun time entry
    fork    begin a new time entry and fork to terminate later
    list    list all the times
    next    begin or end the entry depending on the contents
    run     begin a new time entry and complete upon termination
    status  show the current status of the times
    total   print the sum of all the times
    verify  verify the validity of the times
    wait    upon termination, complete the begun time entry

Options available are:
   -fail	fail if there are any invalid time entries
   -help	print this usage text for track
   -quiet	do not print any informative messages
`)
}

func Verify() error {
	return nil
}

func Status() error {
	return nil
}

func List() error {
	return nil
}

func Total() error {
	f, err := os.Open(pathArg)
	if err != nil {
		return err
	}
	defer f.Close()

	entries, err := readEntries(f, true)
	if err != nil {
		if ferr, ok := err.(*FormatError); ok {
			if !ferr.JustIncomplete() && failFlag {
				return err
			}
			fmt.Fprintf(os.Stderr, "Warning: %s\n", ferr)
		} else {
			return err
		}
	}

	var sum time.Duration
	for _, entry := range entries {
		sum += duration(entry[0], entry[1])
	}

	return nil
}

func Begin() error {
	f, err := os.OpenFile(pathArg, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	return beginEntry(f, failFlag)
}

func Next() error {
	f, err := os.OpenFile(pathArg, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = readEntries(f, false)
	f.Seek(0, 0)
	if err != nil {
		if ferr, ok := err.(*FormatError); ok {
			if ferr.LastIsBad {
				return endEntry(f, failFlag)
			} else if !failFlag {
				return beginEntry(f, failFlag)
			}
		}
		return err
	} else {
		return beginEntry(f, failFlag)
	}
}

func End() error {
	f, err := os.OpenFile(pathArg, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	return endEntry(f, true)
}

func Clean() error {
	return nil
}

func Run() error {
	err := Begin()
	if err != nil {
		return err
	}
	return Wait()
}

// Wait blocks until it receives a signal from the operating system, at which
// it completes the entry in path and exits. If the signal is the Kill signal,
// i.e. SIGKILL, then we exit right away.
func Wait() error {
	c := make(chan os.Signal, 1)
	signal.Notify(c)
	inform("WAIT")
	sig := <-c
	if sig == os.Kill {
		os.Exit(1)
	}
	return End()
}

func Fork() error {
	err := Begin()
	if err != nil {
		return err
	}

	inform("FORK")
	cmd := exec.Command(os.Args[0], "wait", pathArg)
	return cmd.Start()
}

// inform prints str if the global var verbose is true.
func inform(str string) {
	if !quietFlag {
		fmt.Println(str)
	}
}

// currentTime returns the current time as a string.
func currentTime() string {
	return time.Now().Format(timeFormat)
}

// readEntries reads all the entries from r and filters the bad ones out if
// filter is true.
//
// If err is not nil, then it could be of the type *FormatError, or it could
// also originate from csv, in which case just treat it as you would any other
// unknown error.
func readEntries(r io.Reader, filter bool) (entries [][]string, err error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1

	entries, err = reader.ReadAll()
	if err != nil {
		return
	}

	var formatErr FormatError
	for i, entry := range entries {
		if len(entry) == 2 {
			formatErr.LastIsBad = false
		} else {
			formatErr.LastIsBad = true
			if formatErr.BadLines == nil {
				formatErr.BadLines = make([]int, 0, 2)
			}
			formatErr.BadLines = append(formatErr.BadLines, i+1)
		}
	}
	if formatErr.BadLines != nil {
		err = &formatErr

		if filter {
			n := len(entries) - len(formatErr.BadLines)
			filtered := make([][]string, n)
			var i int
			for _, entry := range entries {
				if len(entry) == 2 {
					filtered[i] = entry
					i++
				}
			}
			entries = filtered
		}
	}

	return
}

func beginEntry(rw io.ReadWriter, fail bool) error {
	_, err := readEntries(rw, false)
	if err != nil {
		if _, ok := err.(*FormatError); fail || !ok {
			return err
		}
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	writer := csv.NewWriter(rw)
	writer.Write([]string{currentTime()})
	writer.Flush()
	inform("BEGIN")
	return nil
}

func endEntry(rw io.ReadWriteSeeker, fail bool) error {
	entries, err := readEntries(rw, false)
	if err != nil {
		if ferr, ok := err.(*FormatError); ok && ferr.LastIsBad {
			if len(ferr.BadLines) > 1 {
				if fail {
					return err
				}
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}

			last := append(entries[len(entries)-1], currentTime())
			rw.Seek(-int64(len(last[0])+1), 2) // rewind the last transaction
			writer := csv.NewWriter(rw)
			writer.Write(last)
			writer.Flush()
			inform("END")
			return nil
		}
		return err
	} else {
		return errors.New("no incomplete entry to end")
	}
}

// spokenList returns the list as a string as it would be written in English.
func spokenList(list []int) string {
	var b bytes.Buffer
	for i, n := 0, len(list); i < n; i++ {
		b.WriteString(fmt.Sprint(list[i]))
		if i < n-2 {
			b.WriteString(", ")
		} else if i < n-1 {
			if n == 2 {
				b.WriteString(" and ")
			} else {
				b.WriteString(", and ")
			}
		}
	}
	return b.String()
}

// duration returns the interpretated duration between two times.
func duration(a, b string) time.Duration {
	atime, _ := time.Parse(timeFormat, a)
	btime, _ := time.Parse(timeFormat, b)
	return atime.Sub(btime)
}
