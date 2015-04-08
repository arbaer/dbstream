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
	"fmt"
	"log"
	"os"
	"time"

	schedConn "../scheduler/conn"
	rep "../../server/lib/reporting"
	"../../server/lib/dbs"
	"../../server/utils"
)

var configFileName = flag.String("config", "", "The configuration file used.")
var name = flag.String("name", "", "The name of the module.")
var serverLoc = flag.String("server", "", "The server location to report the status back.")

type Config struct {
	XMLName       xml.Name `xml:"config"`
	CheckInterval int      `xml:"checkInterval,attr"`
	SchedCfg      schedConn.SchedulerCfg
	Dbs           dbs.DbsConn `xml:"dbs"`
	Tables        []TableCfg  `xml:"tables>table"`
}

type TableCfg struct {
	XMLName xml.Name `xml:"table"`
	Name    string   `xml:"name,attr"`
}

type watchTable struct {
	name         string
	partTable    string
	maxTimestamp int64
	token        string
	lastImport   time.Time
}

type extImport struct {
	tables []*watchTable
}

func readConfig() (cfg Config) {
	//decode the xml config file
	cfgFile, err := os.Open(*configFileName)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}
	decode := xml.NewDecoder(cfgFile)
	err = decode.Decode(&cfg)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}
	cfgFile.Close()
	return
}

func main() {
	log.SetFlags(19)
	flag.Parse()
	cfg := readConfig()

	err := rep.Configure(*serverLoc)
	if err != nil {
		log.Fatalf("ERROR: configure reporting error: %v\n", err)
	}

	err = schedConn.Configure(cfg.SchedCfg)
	if err != nil {
		log.Fatalf("ERROR: configure scheduler error: %v\n", err)
	}

	err = dbs.Configure(cfg.Dbs)
	if err != nil {
		log.Fatalf("ERROR: configure TicketDB error: %v\n", err)
	}

	tblLookup := make(map[string]dbs.DbsTable)
	for _, tbl := range dbs.GetTables() {
		tblLookup[tbl.Tablename] = tbl
	}

	var e extImport
	e.tables = make([]*watchTable, len(cfg.Tables))
	for i, cfgTbl := range cfg.Tables {
		tbl, ok := tblLookup[cfgTbl.Name]
		if !ok {
			log.Fatalf("ERROR: configured table %s not found.", cfgTbl.Name)
		}
		newTbl := &watchTable{tbl.Tablename, tbl.PartTablename, 0, "", time.Now()}
		e.tables[i] = newTbl
		go newTbl.Watch(cfg.CheckInterval)
	}
	e.printStatus()
	for {
		select {
		case <-time.After(1 * time.Second):
			rep.SendReport(*name, e.Status())
		}
	}
}

func (e extImport) Status() (status map[string]string) {
	status = make(map[string]string)
	for _, tbl := range e.tables {
		status[tbl.name] = fmt.Sprintf("table:%s partTable:%s maxTs:%d", tbl.name, tbl.partTable, tbl.maxTimestamp)
	}
	return status
}

func (e extImport) printStatus() {
	log.Println("ExternalInput initialized with the following tables")
	for n, s := range e.Status() {
		log.Printf("%s: %s\n", n, s)
	}
}

func (t *watchTable) Watch(checkInterval int) {
	t.maxTimestamp = dbs.GetMaxTimestampForTable(t.partTable)
	outp := utils.IOWindow{t.name, t.partTable, 0, 0, true}
	reply := schedConn.RegisterJobSeries(t.name, "", nil, []utils.IOWindow{outp}, t.maxTimestamp, 0)
	t.token = reply.RegToken
	for {
		time.Sleep(time.Duration(checkInterval) * time.Second)
		newts := dbs.GetMaxTimestampForTable(t.partTable)
		if newts > t.maxTimestamp {
			_, execReply := schedConn.SendAndWaitForJob(t.name, t.token, newts)
			_ = schedConn.SendJobDone(t.name, execReply.JobToken, time.Since(t.lastImport), 0)
			t.lastImport = time.Now()
		}

	}
}
