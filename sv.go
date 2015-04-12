// svctl
// Copyright (C) 2015 Karol 'Kenji Takahashi' Woźniak
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
// DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE
// OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package main

import "time"

// svPid Parses process PID from sv status string.
// Returns `0` if process is not running.
func svPid(status []byte) uint {
	pid := uint(status[15])
	pid <<= 8
	pid += uint(status[14])
	pid <<= 8
	pid += uint(status[13])
	pid <<= 8
	pid += uint(status[12])
	return pid
}

// svStatus Parses process state from sv status string.
func svStatus(status []byte) string {
	switch status[19] {
	case 0:
		return "STOPPED"
	case 1:
		return "RUNNING"
	case 2:
		return "FINISHING"
	default:
		return "UNKNOWN"
	}
}

// svCheck Checks whether process already entered desired state
// after sending it the control action.
func svCheck(action, status []byte, start uint64) bool {
	for _, a := range action {
		pid := svPid(status)
		switch a {
		case 'x':
			//TODO
		case 'u':
			if pid == 0 || status[19] != 1 {
				return false
			}
			//TODO: !checkscript():return false
		case 'd':
			if pid != 0 || status[19] != 0 {
				return false
			}
		case 't', 'k':
			if pid == 0 && status[17] == 'd' {
				break
			}
			time := svTime(status)
			if start > time || pid == 0 || status[18] != 0 { //TODO: ||!checkscript()
				return false
			}
		case 'o':
			time := svTime(status)
			if (pid == 0 && start > time) || (pid != 0 && status[17] != 'd') {
				return false
			}
		case 'p':
			if pid != 0 && status[16] == 0 {
				return false
			}
		case 'c':
			if pid != 0 && status[16] != 0 {
				return false
			}
		}
	}
	return true
}

// svCheckControl Checks whether we should send a control action.
// We should not when process recently got ONCE or TERM.
func svCheckControl(action, status []byte) bool {
	return status[17] != action[0] || (action[0] == 'd' && status[18] != 1)
}

// svTimeMod Is a time shift constant used by sv (copied from sv sources).
const svTimeMod = 4611686018427387914

// svTime Parses time from sv status string.
// It is a diff from the last start/stop.
func svTime(status []byte) uint64 {
	time := uint64(status[0])
	time <<= 8
	time += uint64(status[1])
	time <<= 8
	time += uint64(status[2])
	time <<= 8
	time += uint64(status[3])
	time <<= 8
	time += uint64(status[4])
	time <<= 8
	time += uint64(status[5])
	time <<= 8
	time += uint64(status[6])
	time <<= 8
	time += uint64(status[7])
	return time
}

// svNow Returns current time shifted with sv constant.
func svNow() uint64 {
	return uint64(svTimeMod + time.Now().Unix())
}
