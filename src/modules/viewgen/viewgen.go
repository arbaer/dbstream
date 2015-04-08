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
package main

import (
	"encoding/xml"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"./jobs"
	schedConn "../scheduler/conn"
	rep "../../server/lib/reporting"
	"../../server/lib/dbs"
)

/*
* viewgen
*
* The viewgen registeres jobs at the scheduler and the scheduler makes sure that a job is only executed if data for all inputs is available up to the time needed by this job. 
* For the viewgen, a job can have multiple inputs but only one output. 
*/

type Config struct {
	XMLName    xml.Name `xml:"config"`
	PartSchema string   `xml:"partitionSchema,attr"`
	SchedCfg   schedConn.SchedulerCfg
	Jobs       []jobs.JobDefinition `xml:"jobs>job"`
	Dbs        dbs.DbsConn          `xml:"dbs"`
}

type viewGen struct {
	name string
	cfg  Config
	jobs []*jobs.JobSeries
}

var configFileName = flag.String("config", "", "The configuration file used.")
var name = flag.String("name", "", "The name of the module.")
var serverLoc = flag.String("server", "", "The server location to report the status back.")

func readConfig() (cfg Config) {
	//decode the xml config file
	cfgFile, err := os.Open(*configFileName)
	if err != nil {
		log.Fatalf("ERROR while reading config: %v\n", err)
	}
	decode := xml.NewDecoder(cfgFile)
	err = decode.Decode(&cfg)
	if err != nil {
		log.Fatalf("ERROR while decoding config: %v\n", err)
	}
	cfgFile.Close()
	return
}

func main() {
	flag.Parse()
	log.SetFlags(19)
	cfg := readConfig()


	sigChan := make(chan os.Signal)
	signal.Notify(sigChan)

	//init ticketDB connection
	err := dbs.Configure(cfg.Dbs)
	if err != nil {
		log.Fatalf("ERROR while configuring DBStream: %v", err)
	}


	var v viewGen
	v.name = *name
	v.cfg = cfg
	v.jobs = make([]*jobs.JobSeries, len(cfg.Jobs))
	defer v.exit()

	err = rep.Configure(*serverLoc)
	if err != nil {
		log.Fatalf("ERROR while configuring reporting: %v", err)
	}

	err = schedConn.Configure(cfg.SchedCfg)
	if err != nil {
		log.Fatalf("ERROR while configuring the scheduler connection: %v", err)
	}

	for i, jd := range cfg.Jobs {
		//register all configured job series
		s := new(jobs.JobSeries)
		s.PartSchema = cfg.PartSchema
		s.Register(jd, cfg.SchedCfg.Ip, cfg.SchedCfg.Port)
		v.jobs[i] = s
	}
	for _, s := range v.jobs {
		//start the exection of jobs
		go s.Run()
	}
	for {
		select {
		case <-time.After(1 * time.Second):
			v.sendStatus()
		case sig := <-sigChan:
			if sig == syscall.SIGKILL || sig == syscall.SIGINT || sig == syscall.SIGTERM {
				v.exit()
			}
		}
	}
}

func (v *viewGen) exit() {
	dbs.Close()
	log.Printf("Viewgen %s is shutting down.\n", v.name)
	os.Exit(0)
}

func (v *viewGen) sendStatus() {
	status := make(map[string]string)
	//log.Println("sending status.....")
	for _, j := range v.jobs {
		status[j.Name] = j.Status()
	}

	_, err := rep.SendReport(v.name, status)
	if err != nil {
		log.Fatal("ERROR: reporting status", err)
	}
}
