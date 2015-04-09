DBStream
========

DBStream is a flexible and easy to use Data Stream Warehouse (DSW) designed and implemented at FTW in Vienna, Austria.
The main purpose of DBStream is to store and analyze large amounts of network monitoring data [1].
But, it might also be applied to data from other application domains like e.g. smart grids, smart cities,
intelligent transportation systems, or any other use case that requires continuous processing of large amounts 
of heterogeneous data data over time.
DBStream is implemented as a middle-ware layer on top of PostgreSQL. Whereas all data processing is done in 
PostgreSQL, DBStream offers the ability to receive, store and process multiple data streams in parallel. As we have
shown in a recently published benchmark study [2], DBStream is at least on par with recent large-scale data processing
frameworks like e.g. Hadoop and Spark. 
In addition, DBStream offers a declarative, SQL-based Continuous Execution Language (CEL) which is highly precise
but yet very flexible and easy to use. Using this novel stream processing language, advanced analytics can be 
programmed to run in parallel and continuously over time, using just a few lines of code.

Further information about DBStream can be found in the following research papers:

[1] Arian Baer, Pedro Casas, Lukasz Golab and Alessandro Finamore
"DBStream: An Online Aggregation, Filtering and Processing System for Network Traffic Monitoring"
[http://dx.doi.org/10.1109/IWCMC.2014.6906426](http://dx.doi.org/10.1109/IWCMC.2014.6906426)
Wireless Communications and Mobile Computing Conference (IWCMC), 2014

[2] Arian Baer, Alessandro Finamore, Pedro Casas, Lukasz Golab, Marco Mellia
"Large-Scale Network Traffic Monitoring with DBStream, a System for Rolling Big Data Analysis"
[http://www.tlc-networks.polito.it/mellia/papers/BigData14.pdf](http://www.tlc-networks.polito.it/mellia/papers/BigData14.pdf)
IEEE International Conference on Big Data (IEEE BigData), 2014


Installation
============

In order to compile the go source code of DBStream you have to install the go language:

```
apt-get install golang
```

DBStream also uses several open source libraries which you have to install in order to compile DBStream.

First you need to create a directory where go code can be downloaded to, e.g.:

```
mkdir ~/go
```

Next you need to export a new environment variable so go knows where to put the code, which at least in bash works like this:

```
export GOPATH=~/go
```

Now you can install the needed libraries with the following command:

```
go get github.com/lxn/go-pgsql
go get github.com/go-martini/martini
go get code.google.com/p/vitess/go/cgzip
```

Now go to the DBStream server directory e.g.:

```
cd ~/source/dbstream/
```

and run the build script there:

```
./build.sh
```

The resulting executables will be placed in the bin/ directory. The main 
executable is called hydra which starts the application server.

```
cd bin/
./hydra --config ../config/serverConfig.xml
```

Please edit the server configuration and add the modules of DBStream you wanna use. A detailed description of 
which modules are available and how they can be used will be added soon.

If you want to get some information about the application server you can run the command remote to monitor and control the server.
This command shows the current status of the application server every second:

```
watch -n 1 ./remote
```

The default config also starts a CopyFile module. You can see that currently no files are beeing imported by checking:

http://localhost:3000/DBSImport



Stay tuned for further documentation.


License
=======

Copyright (C) 2013 - 2015 - FTW Forschungszentrum Telekommunikation Wien GmbH (www.ftw.at)

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License, version 3,
as published by the Free Software Foundation.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.

Main Author: Arian Baer (baer _at_ ftw.at)


For other licensing options please contact: baer _AT_ ftw.at
