Monitor
========

A utility tool to watch each CPU core utilization, as well as memory and disk for process.


## Dependency

You need install go SDK

## How to install

 go build monitor.go

## Usuage

./monitor [--extraInfo=True] [--logFile=log.txt] program1 program2 ...

### Examples:

	- Just watch cpu

		'./monitor'

	- Watch cpu and process

		'./monitor firefox'

	- Watch cpu, process, disk, memory

		'./monitor --extraInfo=True'

## Output

It will output below format txt:

date time | NCPU Total(All) Idle(All) Used(All), Total(cpu0) Idel(cpu0) Used(cpu0), ... | ProgramName PID Used 