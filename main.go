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
	"fmt"
	"io"
	"os"
	"time"
)

const defaultTimesFile = "TIMES.csv"

var which = map[string]func(string) error{
	"verify": Verify,
	"status": Status,
	"list":   List,
	"total":  Total,
	"begin":  Begin,
	"end":    End,
	"next":   Next,
	"run":    Run,
	"wait":   Wait,
	"fork":   Fork,
}

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

func main() {
	var path = defaultTimesFile
	var command = Status

	n := len(os.Args)
	if n > 1 {
		command = which[os.Args[1]]
		if command == nil || n > 3 {
			Help()
			return
		}

		if n == 3 {
			path = os.Args[2]
		}
	}

	err := command(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func Help() {
	fmt.Printf(`Usage: tracker [command [file]]

The default command is:
	tracker status %s

Commands available are:
    status  show the current status of the times
    list    list all the times
    total   print the sum of all the times
    begin   begin a new time entry
    end     complete the begun time entry
    next    begin or end the entry depending on the contents
    run     begin a new time entry and complete upon termination
    wait    upon termination, complete the begun time entry
    fork    begin a new time entry and fork to terminate later
    verify  verify the validity of the times
`, defaultTimesFile)
}

func Verify(path string) error { return nil }

func Status(path string) error { return nil }

func List(path string) error { return nil }

func Total(path string) error { return nil }

func Begin(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	return beginEntry(f, true)
}

func Next(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = readEntries(f, false)
	f.Seek(0, 0)
	if err != nil {
		if ferr, ok := err.(*FormatError); ok {
			if ferr.LastIsBad {
				return endEntry(f, true)
			} else if true { // TODO: force
				return beginEntry(f, true)
			}
		}
		return err
	} else {
		return beginEntry(f, true)
	}
}

func End(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	return endEntry(f, true)
}

func Run(path string) error {
	err := Begin(path)
	if err != nil {
		return err
	}
	return Wait(path)
}

func Wait(path string) error { return nil }

func Fork(path string) error { return nil }

// currentTime returns the current time as a string.
func currentTime() string {
	return time.Now().String()
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
	}

	return
}

func beginEntry(rw io.ReadWriter, force bool) error {
	_, err := readEntries(rw, false)
	if err != nil {
		if _, ok := err.(*FormatError); !(force || ok) {
			return err
		}
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	writer := csv.NewWriter(rw)
	writer.Write([]string{currentTime()})
	writer.Flush()
	fmt.Println("BEGIN")
	return nil
}

func endEntry(rw io.ReadWriteSeeker, force bool) error {
	entries, err := readEntries(rw, false)
	if err != nil {
		if ferr, ok := err.(*FormatError); ok && ferr.LastIsBad {
			if len(ferr.BadLines) > 1 {
				if !force {
					return err
				}
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}

			last := append(entries[len(entries)-1], currentTime())
			rw.Seek(-int64(len(last[0])+1), 2) // rewind the last transaction
			writer := csv.NewWriter(rw)
			writer.Write(last)
			writer.Flush()
			fmt.Println("END")
			return nil
		}
		return err
	} else {
		return errors.New("no incomplete entry to end")
	}
}

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
