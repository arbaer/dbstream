Copyright (C) 2013 - 2015 - FTW Forschungszentrum Telekommunikation Wien GmbH (www.ftw.at) 

This file describes how to use the Mplane Authorized Transfer via HTTP (MATH). There are two executables provided.

math_probe -- handles the probe part, i.e. it provides the ability to download files following the MATH protocol.
math_repo  -- handles the download of files and so far only implements a plain copy mode.

math_probe.xml -- is the configuration file for the probe. The most important part is the directory XML attribute to configure the directory to copy.
math_repo.xml -- is the configuration file for the repository. The most important part here is the outDir XML attribute of the fileHandlerConfig XML element, which is used as the target for the files being copied.


To start the transfer of file using MATH you first need to start configure and start the repository part:

```
./math_repo
```

Then, in another shell or on the other machine, the data is coming from you need to start the probe part:

```
./math_probe --repoUrl "localhost:3000"
```

Pleaxse note that for the moment you need to give the URL of the repository as a commandline argument. LAter on this will be handled by the mPlane supervisor.

Please note, that the math_repo can be used in combination with the DBStream hydra server process or as a stand alone executable.


