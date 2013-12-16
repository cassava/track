// Copyright (c) 2013, Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

// Track is a program to track the time you spend on a project by storing
// the start and end times in a CSV file.
//
// The aim of track is to make it easy to manage your time.
package main

import (
	"os"
)

const which = map[string]func(string){
	"verify": verify,
	"status": status,
	"list":   list,
	"total":  total,
	"begin":  begin,
	"end":    end,
	"run":    run,
	"wait":   wait,
	"fork":   fork,
}

func main() {
	var path = "TIMES.csv"
	var command = status

	n := len(os.Args)
	if n > 1 {
		command = which[os.Args[1]]
		if command == nil || n > 3 {
			help()
			return
		}

		if n == 3 {
			path = os.Args[2]
		}
	}

	command(path)
}

func help() {}

func verify(path string) {}

func status(path string) {}

func list(path string) {}

func total(path string) {}

func begin(path string) {}

func end(path string) {}

func run(path string) {}

func wait(path string) {}

func fork(path string) {}
