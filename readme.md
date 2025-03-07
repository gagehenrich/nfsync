# nfsync
Whether migrating vast datasets or keeping distributed storage in sync, nfsync delivers reliability, speed, and transparency
 
## Pre-requisites
go 1.20+ installed

### Clone this repo
`git clone https://github.com/gagehenrich/nfsync.git && cd nfsync`

### Compile 
`go build -o nfsync` 

### Usage
`./nfsync [ optional: --threads <num_threads> ] <src> <dst>` 
**NOTE:** threads should depend on your hardware. Deault threads is 64.

### Logging
nfsync provides detailed logs for every operation, enabling thorough auditing of large-scale file transfers. 

ex: 
```sh
$ ./nfsync --threads 32 /NAS/LAB/VOD cephfs/VOD
2024/08/15 07:18:01 | PROC | INFO | Allocating number of threads: 32
2024/08/15 07:18:01 | FILE | INFO | Indexing files on /NAS/LAB/VOD/
2024/08/15 07:18:01 | COPY | INFO | /NAS/LAB/VOD/ConnectorFsIsWritableCheck copied to cephfs/VOD/ConnectorFsIsWritableCheck
2024/08/15 07:18:01 | COPY | INFO | /NAS/LAB/VOD/describe/INF/33329178.ism.descr copied to cephfs/VOD/describe/INF/33329178.ism.descr
2024/08/15 07:18:01 | COPY | INFO | /NAS/LAB/VOD/describe/INF/33329193.ism.descr copied to cephfs/VOD/describe/INF/33329193.ism.descr
2024/08/15 07:18:01 | COPY | INFO | /NAS/LAB/VOD/describe/INF/33314439.ism.descr copied to cephfs/VOD/describe/INF/33314439.ism.descr
2024/08/15 07:18:01 | COPY | INFO | /NAS/LAB/VOD/describe/INF/33311679.ism.descr copied to cephfs/VOD/describe/INF/33311679.ism.descr
2024/08/15 07:18:01 | FILE | INFO | Skipping: cephfs/VOD/describe/INF/33329032.ism.descr already exists and matches the source file
...
```
