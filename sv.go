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
