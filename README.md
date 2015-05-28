
[![mPlane](http://www.ict-mplane.eu/sites/default/files//public/mplane_final_256x_0.png)](http://www.ict-mplane.eu/)

![alt text](https://github.com/arbaer/dbstream/blob/master/dbs_logo.png "DBStream logo")


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

If you are using DBStream for any research purpose we would highly appreciate if you reference [2].


Installation
============

## Note for older Ubuntu Versions

DBStream and the used libraries assume that you are using golang version 1.2.x. Therefore, for older versions of Ubuntu like e.g. 12.04 you might follow the instructions in this guide: 

[http://www.tuomotanskanen.fi/installing-go-1-2-on-ubuntu-12-04-lts/](http://www.tuomotanskanen.fi/installing-go-1-2-on-ubuntu-12-04-lts/)

## General Installation Instructions

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

The default config also starts a CopyFile module. You can see that currently no files are being imported by checking:

http://localhost:3000/DBSImport



Setting up Postgres as a DBStream Backend
=========================================

The first step is to install PostgreSQL. We where using versions from up to 8.4 for DBStream, but rather recommend to use newer versions, like e.g. 9.3. On ubuntu you can install PostgreSQL with the following command:


```
apt-get install postgresql-9.3
```

The next step is to create an operating system and database user for DBStream. From now on, we will assume that this user is called "dbs_test" but you can choose any other user name, just make sure that all parts of the configuration are adapted as well. This user has to be a postgres superuser.

```
sudo useradd -s /bin/bash -m dbs_test
```

Now you have to create a database with the name of that user, please note, that this database will also be used to store all data imported to and processed with DBStream.

```
sudo su - postgres           # change to the postgres user
createuser -P -s dbs_test    # create new user with superuser rights and set password
createdb dbs_test            # create a database with the same name
exit                         # close the postgres user session

```

DBStream uses two tablespaces to store data on disk, namely data0 and view0. For testing purposes, we will locate them in the home folder of the dbs_test user, but in a real setup you probably want to set them to a large RAID-10 storage array.

```
sudo mkdir /home/dbs_test/dbs_ts0             # create data0
sudo chown postgres /home/dbs_test/dbs_ts0    # This directory must be accessable by the postgres system user
```

Now the newly created DBStream database needs to be initialized. Therefore, change to the test directory and login into the database you just created:

```
cd test
psql dbs_test        # Please note that you need to login with a database superuser, so you might want to change to the dbs_test user first.
```

If you log correctly into the database you should see something like this:

```
psql (9.3.6)
Type "help" for help.

dbs_test=# 
```

Now please run the following command to initialize some DBStream internal tables.

```
\i initialize.sql
```

If all steps from this part completed successfully you can go on and start DBStream for the first time!


## Starting DBStream

First we need to have all DBStream executables available in the test directory.

```
cd test
ln -s ../bin/* .  # make sure that the build command from the previous part was successful.
```

Now you should see the executables in this directory (e.g. hydra, math_probe, math_repo, scheduler and remote). For this example it is the best to open three shells. In the first shell we will run *dbstream*, in the second we will run the *import source* and the third will be used for *monitoring* DBStream.

In the *monitoring* shell run the following command:

```
cd dbstream/test
watch -n 1 ./remote
```


In the *dbstream* shell execute the following command:

```
cd dbstream/test
./hydra --config sc_tstat.xml
```

In the *import source* run the following command:

```
cd dbstream/test
./math_probe --config math_prob.xml
```

If all went well, you should now be able to log into postgres:

```
select * from example_log_tcp_complete; 
```

To cleanup the tables and run the example import again, inside postgres execute the following command:

```
select dbs_drop_table('example_log_tcp_complete'); select dbs_drop_table('tstat_test');
```

and in the shell run:

```
rm -rf /tmp/target/
```

Stay tuned for even further documentation.


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
