/* 
* Copyright (C) 2013 - 2015 - FTW Forschungszentrum Telekommunikation Wien GmbH (www.ftw.at)
*
* This program is free software: you can redistribute it and/or modify
* it under the terms of the GNU Affero General Public License, version 3,
* as published by the Free Software Foundation.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
* GNU Affero General Public License for more details.
*
* You should have received a copy of the GNU Affero General Public License
* along with this program. If not, see <http://www.gnu.org/licenses/>.
*
* Author(s): Arian Baer (baer _at_ ftw.at)
*
*/
package utils

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"time"
)

type Module struct {
	Name           string `xml:"name,attr"`
	Executable     string `xml:"exec,attr"`
	Config         string `xml:",innerxml"`
	Arguments      string `xml:"args,attr"`
	RestartPenalty int    `xml:"restartPenalty,attr"`
	cmd            *exec.Cmd
	restartCount   int
	startTime      time.Time
	Running        bool
	confFile       *os.File

	Stats *ModStat
}

type ModStat struct {
	ReportTime time.Time
	Values     map[string]string
}

func (m *Module) Status() (status string) {
	return m.StatusFilt(".*")
}

func (m Module) StatusFilt(filter string) (status string) {
	pid := "Not available"
	if m.cmd != nil && m.cmd.Process != nil {
		pid = fmt.Sprintf("%d", m.cmd.Process.Pid)
	}
	status = fmt.Sprintf("PID: %s StartTime: %s restarted: %d", pid, m.startTime.String(), m.restartCount)
	if m.Stats != nil {
		status = fmt.Sprintf("%s\n\tDetailed Status was last updated: %v", status, m.Stats.ReportTime)
		names := make([]string, len(m.Stats.Values))
		i := 0
		for name, _ := range m.Stats.Values {
			found, err := regexp.MatchString(filter, name)
			if err != nil {
				log.Println(err)
			}
			if found {
				names[i] = name
				i++
			}
		}
		names = names[:i]
		sort.Strings(names)
		for _, n := range names {
			status = fmt.Sprintf("%s\n\t%s: %s", status, n, m.Stats.Values[n])
		}
	}
	return status
}

func (module *Module) Start(serverLoc string, logChan chan string, exitChan chan Module) (err error) {
	//check if executable is available
	_, err = exec.LookPath(module.Executable)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	if module.Config != "" {
		//create config file
		tempF, err := ioutil.TempFile("", "puppetMasterConf_")
		if err != nil {
			return err
		}
		_, err = tempF.WriteString(module.Config)
		if err != nil {
			return err
		}
		err = tempF.Close()
		if err != nil {
			return err
		}
		module.confFile = tempF

		//create the command

		cmd = exec.Command(module.Executable, "--config", tempF.Name(),
			"--server", serverLoc,
			"--name", module.Name,
			module.Arguments)
	} else {
		cmd = exec.Command(module.Executable, "--server", serverLoc, module.Arguments)
	}
	if err != nil {
		return err
	}

	module.cmd = cmd

	//register the logging
	errReader, err := cmd.StderrPipe()
	go module.registerLog(errReader, logChan)

	//start the module and wait for it to exit
	module.startTime = time.Now()
	err = cmd.Start()
	if err != nil {
		return err
	}
	go module.waitForExit(exitChan)

	module.Running = true
	return nil
}

func (m *Module) registerLog(errReader io.ReadCloser, logChan chan string) {
	reader := bufio.NewReader(errReader)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Print("LogError for Module "+m.Name+": ", err)
			return
		}
		logChan <- m.Name + ": " + line
	}
}

func (m *Module) waitForExit(exitChan chan Module) {
	err := m.cmd.Wait()
	if err != nil {
		log.Println(err)
	}
	exitChan <- *m
}

func (module *Module) Restart(serverLoc string, logChan chan string, exitChan chan Module) {
	if time.Since(module.startTime) < 60*time.Second {
		log.Printf("Module %v crashed to early applying wait penalty.\n", module.Name)
		time.Sleep(time.Duration(module.RestartPenalty) * time.Second)
	} else {
		time.Sleep(1 * time.Second)
	}
	if !module.startTime.IsZero() {
		log.Printf("Module %v restarted for the %v time, after: %v\n", module.Name, module.restartCount, time.Since(module.startTime))
	}
	module.restartCount++
	module.Start(serverLoc, logChan, exitChan)
}
