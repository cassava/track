// Copyright (c) 2013, Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

// Track is a program to track the time you spend on a project by storing
// the start and end times in a CSV file.
//
// The aim of track is to make it easy to manage your time.
package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/goulash/util"
)

const defaultTimesFile = "TIMES.csv"

var which = map[string]func(string){
	"verify": Verify,
	"status": Status,
	"list":   List,
	"total":  Total,
	"begin":  Begin,
	"end":    End,
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
		return fmt.Sprint("contains invalid entries on line(s)", e.BadLines...)
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
		fmt.Fprintf("Error: %s\n", err)
		os.Exit(1)
	}
}

func Help() {
	fmt.Printf(`Usage: tracker [command [file]]

The default command is:
	tracker status %s

Commands available are:
    verify  verify the validity of the times
    status  show the current status of the times
    list    list all the times
    total   print the sum of all the times
    begin   begin a new time entry
    end     complete the begun time entry
	next	begin or end the entry depending on the contents
    run     begin a new time entry and complete upon termination
    wait    upon termination, complete the begun time entry
    fork    begin a new time entry and fork to terminate later
`, defaultTimesFile)
}

func Verify(path string) error {}

func Status(path string) error {}

func List(path string) error {}

func Total(path string) error {}

func Begin(path string) error {
	_, err := util.IsFileExists(path)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = verify(f); err != nil {
		return err
	}
	begin(f)
	return nil
}

func begin(f *File) {
	w := csv.NewWriter(f)
	w.Write([]string{currentTime()})
	w.Flush()
}

func Next(path string) error {
	_, err := util.IsFileExists(path)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = verify(f); err != nil {
		if ferr, ok := err.(*FormatError); ok {
			if ferr.JustIncomplete() {
				end(f) // then we complete it
				return nil
			}
		}
		return err
	}
	begin(f)
	return nil
}

func End(path string) error {}

func end(f *File) {
	// TODO
	w := csv.NewWriter(f)
}

func Run(path string) error {
	Begin(path)
	Wait(path)
}

func Wait(path string) error {}

func Fork(path string) error {}

// currentTime returns the current time as a string.
func currentTime() string {
	return time.Now().String()
}

// verify reads through all records from the Reader r, and returns a FormatError
// if any of the entries are invalid or incomplete.
func verify(r io.Reader) (count int, err error) {
	v := csv.NewReader(r)
	v.FieldsPerRecord = -1

	var formatErr FormatError
	for {
		r, err := v.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		count++
		if len(r) == 2 {
			formatErr.LastIsBad = false
		} else {
			formatErr.LastIsBad = true
			if formatErr.BadLines == nil {
				formatErr.BadLines = make([]int, 0, 2)
			}
			append(formatErr.BadLines, count)
		}
	}

	if formatErr.BadLines != nil {
		return &formatErr
	}

	return nil
}
