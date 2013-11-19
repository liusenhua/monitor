CPU Utilization Test Summary
============================

## Pre-Requisites

### Machine: one 2-core machine and 8-core machine

### OS: Linux Debian/Ubuntu 12.04

### Test Programs:
    -   Monitor.go: A monitor program use below algorithms to calculate each cpu core, the total cpu,
        and process of cpu utilization
        +   A algorithm: (totalCPU- totalIdle) / totalCPU
        +   B algorithm: uint64(ncpu) * (ucpu+scpu) / totalCPU
        +   C algorithm: (ucpu+scpu) / totalCPU

    -   calPI.go: A test program to calculate math PI value, it will make cpu run with 100% utilization.
        It support specify how many cpu core it used when computing.

## Test Work flow

    -   Execute “./calPI –ncpu=1” to calculate PI with specifying only one cpu core
    -   Execute “./monitor calPI” to monitor CPU data

    We run many cycles, in each cycle will increase the number of cpu core.

## Results
    -  (totalCPU- totalIdle)/totalCPU solution is correct.

        1.  B algorithm: uint64(ncpu)*(ucpu+scpu)/ totalCPU, I can understand the multiple cause the range
            to go from 0…800% on an 8CPU machine, but that doesn’t make any sense, on the contrary, it make me confused. Because I got 196% on 2-core machine and 793% on 8-core machine when all cpu core are running out.

        2.  Both of algorithm A and C are correct, they are two different perspectives: A shows cpu core 
            usage, but C shows process usage. See “8-cpu-with-others.txt”.

    - Observations from the test

        1.  To leverage go routines concurrent capability, we need call runtime.GOMAXPROCS(np)
            explicitly to specify how many cpu core. See “without-GOMAXPROC.txt’ and “has-GOMAXPROC.txt”

        2.  The go routines in calPI.go can make each core run 100% usage.

        3.  The performance of calPI.go goes up when ncpu increase. See ‘calPI-result.txt’
