In order to run a DBStream demo, you need to previously follw these steps: 

1) install vagrant version 1.7.2 from http://www.vagrantup.com/downloads.html

2) add the ubuntu/trusty64 box:
$$ vagrant box add ubuntu/trusty64

3) start the virtual machine:
$$ vagrant up

4) connect to the virtual machine:
$$ vagrant ssh

5) enjoy DBStream by continuing [here](https://github.com/arbaer/dbstream#starting-dbstream)!

In case you want to recompile DBStream please set the GOPATH environment variable to and execute the build.sh single threaded:

```
export GOPATH=~/go
./build.sh single
```
