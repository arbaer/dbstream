#!/usr/bin/env bash
set -x

# set locale
sudo su << 'EOF'
export LANGUAGE="en_US.UTF-8"
echo 'LANGUAGE="en_US.UTF-8"' >> /etc/default/locale
echo 'LC_ALL="en_US.UTF-8"' >> /etc/default/locale
locale-gen de_AT.utf8
dpkg-reconfigure locale
exit
EOF

# install golang
sudo apt-get install -y golang

# intall git to clone DBStream repository
sudo apt-get install -y git

# install required software packages required bycgzip
sudo apt-get -y install mercurial
sudo apt-get -y install pkg-config
sudo apt-get -y install zlib1g-dev

# export gopath
mkdir ~/go
export GOPATH=~/go

# install libraries for DBStream
go get github.com/lxn/go-pgsql
go get github.com/go-martini/martini
go get code.google.com/p/vitess/go/cgzip

# download DBStream
mkdir src
cd src
git clone https://github.com/arbaer/dbstream/

# add postgres apt repository
sudo su <<'EOF'
echo 'deb http://apt.postgresql.org/pub/repos/apt/ trusty-pgdg main' > /etc/apt/sources.list.d/pgdg.list
exit
EOF
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | \
  sudo apt-key add -
sudo apt-get update

# install PostgreSQL
sudo apt-get install -y postgresql-9.4

# create postgresql cluster and user
sudo useradd -s /bin/bash -m dbs_test
sudo su postgres <<'EOF'
pg_createcluster 9.4 dbs_test
pg_ctlcluster 9.4 dbs_test start
createdb dbs_test
exit
EOF

# create tablespace for DBStram
sudo mkdir /home/dbs_test/dbs_ts0
sudo chown postgres /home/dbs_test/dbs_ts0 

# initialize tables required by dbstream
sudo su postgres << 'EOF'
cd /home/vagrant/src/dbstream/test/
psql dbs_test -c "\i initialize.sql"
psql -c "alter user dbs_test password 'test'"
exit
EOF

export GOPATH=~/go
# compile DBStream
cd /home/vagrant/src/dbstream
./build.sh single
set +x
