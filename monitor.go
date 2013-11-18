package main

import "flag"
import "fmt"
import "strings"
import "strconv"
import "bytes"
//import "io"
import "io/ioutil"
//import "bufio"
import "os"
import "os/exec"
import "os/signal"
import "syscall"
import "time"
import "unicode"
import "log"
import "regexp"
import "reflect"
//-----------------Utility function -------------------

type cpuStat struct {
	// Values are USER_HZ (1/100 second)
	cpuTotal                                                            uint64
	cpuUser, cpuNice, cpuSystem, cpuIdle, cpuIowait, cpuIrq, cpuSoftIrq uint64

	ctxtSwitches uint64
	interrupts   uint64
}

func procStat() (values []cpuStat, err error) {
	values = make([]cpuStat, 0)
	if stat, err := ioutil.ReadFile("/proc/stat"); err == nil {
		var tmp = cpuStat{}
		var cpuTotal *cpuStat
		for _, line := range bytes.Split(stat, []byte{'\n'}) {
			fields := strings.Fields(string(line))
			if len(fields) < 2 {
				continue
			}
			
			if match, _ := regexp.MatchString("^cpu[0-9]*", fields[0]); match == true {
				s := cpuStat{}
				for i := 1; i < 10; i++ {
					if v, err := strconv.ParseUint(fields[i], 10, 64); err == nil {
						s.cpuTotal += v
					}
				}


				s.cpuUser, _ = strconv.ParseUint(fields[1], 10, 64)
				s.cpuNice, _ = strconv.ParseUint(fields[2], 10, 64)
				s.cpuSystem, _ = strconv.ParseUint(fields[3], 10, 64)
				s.cpuIdle, _ = strconv.ParseUint(fields[4], 10, 64)
				s.cpuIowait, _ = strconv.ParseUint(fields[5], 10, 64)
				s.cpuIrq, _ = strconv.ParseUint(fields[6], 10, 64)
				s.cpuSoftIrq, _ = strconv.ParseUint(fields[7], 10, 64)
				
				if fields[0] == "cpu" {
					cpuTotal = &s
				}
				values = append(values, s)
			} else if fields[0] == "intr" {
				tmp.interrupts, _ = strconv.ParseUint(fields[1], 10, 64)
			} else if fields[0] == "ctxt" {
				tmp.ctxtSwitches, _ = strconv.ParseUint(fields[1], 10, 64)
			}
		}
		if cpuTotal != nil {
			cpuTotal.interrupts = tmp.interrupts
			cpuTotal.ctxtSwitches = tmp.ctxtSwitches
		}

	}

	return
}

// procNCPU returns the number of CPUs on the system
func procNCPU() (ncpu uint64) {
	if stat, err := ioutil.ReadFile("/proc/stat"); err == nil {
		fields := strings.Fields(string(stat))
		for _, f := range fields {
			if len(f) >= 4 && f[0:3] == "cpu" && unicode.IsDigit(rune(f[3])) {
				ncpu++
			}
		}
	}
	return
}

type memInfo struct {
	MemTotal uint64
	MemFree  uint64
	MemUsed  uint64
	Buffers  uint64
	Cached   uint64

	MemUsage uint64
}

func procMeminfo() (mi memInfo) {
	if stat, err := ioutil.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range bytes.Split(stat, []byte{'\n'}) {
			fields := strings.Fields(string(line))
			if len(fields) < 2 {
				continue
			}

			switch fields[0] {
			case "MemTotal:":
				mi.MemTotal, _ = strconv.ParseUint(fields[1], 10, 64)
				mi.MemTotal *= 1024

			case "MemFree:":
				mi.MemFree, _ = strconv.ParseUint(fields[1], 10, 64)
				mi.MemFree *= 1024

			case "Buffers:":
				mi.Buffers, _ = strconv.ParseUint(fields[1], 10, 64)
				mi.Buffers *= 1024

			case "Cached:":
				mi.Cached, _ = strconv.ParseUint(fields[1], 10, 64)
				mi.Cached *= 1024

			}

			mi.MemUsed = mi.MemTotal - mi.MemFree - mi.Buffers - mi.Cached
			mi.MemUsage = (100 * (mi.MemTotal - mi.MemFree - mi.Buffers - mi.Cached)) / mi.MemTotal

		}
	}
	return
}

type netStat struct {
	rcvBytes, rcvPackets, rcvErrors uint64
	sndBytes, sndPackets, sndErrors uint64
}

func procNetStat(iface string) (s netStat) {
	if stat, err := ioutil.ReadFile("/proc/net/dev"); err == nil {
		for _, line := range bytes.Split(stat, []byte{'\n'}) {
			fields := strings.SplitN(string(line), ":", 2)
			if len(fields) < 2 {
				continue
			}

			rest := strings.Fields(fields[1])
			if strings.TrimSpace(fields[0]) == iface {
				s.rcvBytes, _ = strconv.ParseUint(rest[0], 10, 64)
				s.rcvPackets, _ = strconv.ParseUint(rest[1], 10, 64)
				s.sndBytes, _ = strconv.ParseUint(rest[8], 10, 64)
				s.sndPackets, _ = strconv.ParseUint(rest[9], 10, 64)
				break
			}
		}
	}
	return
}

type taskStatus struct {
	utime uint64
	stime uint64

	// Values are in kB
	vmSize uint64
	vmRSS  uint64

	volCtxtSwitches   uint64
	involCtxtSwitches uint64
}

func procTaskStatus(pid int) (ts taskStatus) {
	if stat, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/status", pid)); err == nil {
		for _, line := range bytes.Split(stat, []byte{'\n'}) {
			fields := strings.Fields(string(line))
			if len(fields) < 2 {
				continue
			}

			switch fields[0] {
			case "VmRSS:":
				ts.vmRSS, _ = strconv.ParseUint(fields[1], 10, 64)
				ts.vmRSS *= 1024

			case "VmSize:":
				ts.vmSize, _ = strconv.ParseUint(fields[1], 10, 64)
				ts.vmSize *= 1024

			case "voluntary_ctxt_switches:":
				ts.volCtxtSwitches, _ = strconv.ParseUint(fields[1], 10, 64)

			case "nonvoluntary_ctxt_switches:":
				ts.involCtxtSwitches, _ = strconv.ParseUint(fields[1], 10, 64)
			}
		}
	}

	if stat, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/stat", pid)); err == nil {
		fields := strings.Fields(string(stat))
		ts.utime, _ = strconv.ParseUint(fields[13], 10, // user mode jiffies with child's
			64)
		ts.stime, _ = strconv.ParseUint(fields[14], 10, // kernel mode jiffies with child's
			64)
	}

	return
}

type diskStatus struct {
	BytesFree, BytesTotal uint64
	InodeFree, InodeTotal uint64
}

func procDiskStat(path string) (ds diskStatus) {
	var buf syscall.Statfs_t
	if err := syscall.Statfs(path, &buf); err == nil {
		bsize := uint64(buf.Bsize)
		ds.BytesFree = buf.Bfree * bsize
		ds.BytesTotal = buf.Blocks * bsize
		ds.InodeFree = buf.Ffree
		ds.InodeTotal = buf.Files
	}
	return
}

//------------------------------------
const (
	terminationTimeout = 15      // seconds
	dirWatcherInterval = 10      // seconds
	statInterval       = 2       // seconds
	cpuNInterval       = 60 * 10 // seconds
	respawnPenalty     = 5       // seconds
)

var (
	ncpu = procNCPU()
)

func readPid(file string) int {
	if pidstr, err := ioutil.ReadFile(file); err == nil {
		if pid, err := strconv.Atoi(string(pidstr)); err == nil {
			return pid
		}
	}
	return -1
}

func Write(path string, sync bool, buf []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(buf)
	if err != nil {
		return err
	}
	if sync {
		return f.Sync()
	}
	return nil
}

func writePid(filename string, pid int) error {
	return Write(filename, true, []byte(strconv.Itoa(pid)))
}

func alive(pid int) bool {
	if err := syscall.Kill(pid, 0); err == nil {
		return true
	}
	return false
}

func getPidFromProcessName(processName string ) int {
    argStr := "ps -A | grep -m1 " + processName + " | awk '{print $1}'" // "-c", "ps -A | grep -m1 firefox | awk '{print $1}'"
    cmd := exec.Command("/bin/sh", "-c", argStr)
    out, err := cmd.Output()

    if err != nil {
        println(err.Error())
        return -1
    }

    if pid, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil {
    	return pid
	}

	return -1
}

var stop = make(chan bool)

func monitorStop() {
	stop <- true
	<-stop
}

type namePidPair struct {
	name string
	pid int
}

func monitor(processNames []string, extraInfo bool) {
	// Switch to effective uid
	euid := syscall.Geteuid()
	syscall.Setreuid(euid, euid)

	fmt.Printf("Node monitor started (running with uid %d)\n", syscall.Getuid())

	targets := make([]namePidPair, 0)
	for i := 0; i < len(processNames); i++ {
		processName := processNames[i]
		pid := getPidFromProcessName(processName)
		if pid != -1 {
			fmt.Printf("Pid %d of \"%s\" is being monitoring...\n", pid, processName)
			targets = append(targets, namePidPair{processName, pid})
		} else {
			fmt.Printf("Can't find process pid for %s!\n", processName)
		}
	}

	headerFields := []string {}

	// Timestamp field
	stampFieldH := "Date Time"
	headerFields = append(headerFields, stampFieldH)
	
	// CPU field
	cpuFieldsH := "NCPU(All) Total(All) Tdle(All) Used(All)"
	for i:= 0; i < (int)(ncpu); i++ {
		cpuFieldsH = cpuFieldsH + fmt.Sprintf(" , Total(cpu%d) Idle(cpu%d) Used(cpu%d)", i, i, i)
	}
	headerFields = append(headerFields, cpuFieldsH)

	// Process field
	processFieldsH := ""
	for i := 0; i < len(targets); i++ {
		if i != 0 {
			processFieldsH = processFieldsH + " ,"
		}
		processFieldsH = processFieldsH + " Name PID Used Used*NCPU" 
	}
	headerFields = append(headerFields, processFieldsH)

	// Print other information, disk, meminfo, netinfo
	if extraInfo {
		ds := procDiskStat("/disk")
		meminfo := procMeminfo()
		//netstat := procNetStat("eth0")
		items := []interface{}{ &ds, &meminfo, /*&netstat*/ }
		for i := 0; i < len(items); i++ {
			extraFieldsH := ""
			elem := reflect.ValueOf(items[i]).Elem()
			typeOfT := elem.Type()
			for j := 0; j < elem.NumField(); j++ {
				extraFieldsH = extraFieldsH + " " + typeOfT.Field(j).Name
			}
			headerFields = append(headerFields, extraFieldsH)
		}
	}

	logHeader(strings.Join(headerFields, " |"))


	// Record the initial cpu and task status
	preStats, _:= procStat()
	preTaskStats := make([]taskStatus, 0)
	for i := 0; i < len(targets); i++ {
		preTs := procTaskStatus(targets[i].pid)
		preTaskStats = append(preTaskStats, preTs)
	}


	statTicker := time.NewTicker(statInterval * 1e9)
	defer statTicker.Stop()

	for {
		select {

		case <-statTicker.C:
			if len(preStats) < 1 {
				continue
			}

			fields := make([]string, 0)

			stats, err:= procStat()
			if err == nil {
				// cpu field
				cpuFields := strconv.FormatUint(ncpu, 10)

				for i := 0; i < len(stats); i++ {

					stat := stats[i]
					preStat := preStats[i]

					dtTotal := stat.cpuTotal - preStat.cpuTotal
					dtIdle := stat.cpuIdle - preStat.cpuIdle
					cpuUsedPercent := 100 * (dtTotal - dtIdle) / dtTotal

					if i != 0 {
						cpuFields = cpuFields + " ,"
					}

					currentField := fmt.Sprintf("%v %v %v%%", dtTotal, dtIdle, cpuUsedPercent)
					cpuFields = cpuFields + " " + currentField
				}
				//fmt.Println(cpuFields)
				fields = append(fields, cpuFields)

				processFields := ""
				for i := 0; i < len(targets); i++ {
					processName := targets[i].name
					pid := targets[i].pid
					if !alive(pid) {
						fmt.Printf("%d of %s died.\n", pid, processName)
					}

					ts := procTaskStatus(pid)
					preTs := preTaskStats[i]
					utime := ts.utime - preTs.utime
					stime := ts.stime - preTs.stime
					processUsed := utime + stime
					processUsedPercent := 100 * processUsed / (stats[0].cpuTotal - preStats[0].cpuTotal)
					processUsedPercentMuli := ncpu * processUsedPercent

					if i != 0 {
						processFields = processFields + " ,"
					}
					processFields = processFields + " " + fmt.Sprintf("%v %v %v %v%% %v%%", processName, pid, processUsed, processUsedPercent, processUsedPercentMuli)
					
					preTaskStats[i] = ts
				}
				//fmt.Println(processFields)
				fields = append(fields, processFields)

				preStats = stats

				// print other information, disk, meminfo, netstat
				if extraInfo {
					ds := procDiskStat("/disk")
					meminfo := procMeminfo()
					//netstat := procNetStat("eth0")
					items := []interface{}{ &ds, &meminfo, /*&netstat*/ }
					for i := 0; i < len(items); i++ {
						extraFields := ""
						elem := reflect.ValueOf(items[i]).Elem()
						for j := 0; j < elem.NumField(); j++ {
							f := elem.Field(j)
							value := f.Interface()
							extraFields = extraFields + " " + fmt.Sprintf("%v", value)
						}
						fields = append(fields, extraFields)
					}
				}

				logStat("| %v", strings.Join(fields, " |"))
			} else {
				panic("Error- Can't read /proc/stat, it is not linux os!")
			}

		case <-stop:
			stop <- true
			break
		}
	}
}

func logHeader(header string) {
	s := fmt.Sprintf("%s\n", header)
	fmt.Printf(s)
	//n, err := fLog.WriteString(s)
	//fLog.Sync()
	//fmt.Printf("Write header %d bytes, err is %s\n", n, err)
}
func logStat(format string, args ...interface{}) {
	if fileLogger != nil {
		fileLogger.Printf(format, args...)
	}
	if stdoutLogger != nil {
		stdoutLogger.Printf(format, args...)
	}
}

var fLog *os.File
var fileLogger *log.Logger
var stdoutLogger *log.Logger
var extraInfo = flag.Bool("extraInfo", false, "Log the extra information(disk, memory, net) if it is specified as True")
var logFile = flag.String("logFile", "log.txt", "Specify monitor log file")
func main() {
	flag.Parse()

	// prepare the file logger and std out logger
	logFlags := log.Ldate | log.Ltime
	fLog, err := os.Create(*logFile)
    if err != nil {
        panic(err)
    }
    defer fLog.Close()


	//fLog.WriteString("Hello!\n")
	fileLogger = log.New(fLog, "", logFlags)
	stdoutLogger = log.New(os.Stdout, "", logFlags)

	fmt.Println("monitor - starting...")


    processNames := make([]string, 0)
    for i := 1; i < len(os.Args); i++ {
    	var arg = os.Args[i]
    	if !strings.HasPrefix(arg, "-") {
    		processNames = append(processNames, arg)
    	}
    }
    go monitor(processNames, *extraInfo)
	
    sigch := make(chan os.Signal)
	signal.Notify(sigch)

	for {
		sig := <-sigch
		switch sig {
		case syscall.SIGTERM:
			fmt.Println("node - syscall.SIGTERM")
			return
		case syscall.SIGINT:
			fmt.Println("node - syscall.SIGINT")
			monitorStop()
			return
		}
	}
}
