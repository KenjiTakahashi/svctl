package main

import "time"

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

const svTimeMod = 4611686018427387914

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

func svNow() uint64 {
	return uint64(svTimeMod + time.Now().Unix())
}
