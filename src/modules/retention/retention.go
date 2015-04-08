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
	"strings"
	"strconv"
	"time"
	
	schedConn "../scheduler/conn"
	"../scheduler/remote"
	rep "../../server/lib/reporting"
	"../../server/lib/dbs"
	"../../server/utils"
)

var configFileName = flag.String("config", "", "The configuration file used.")
var name = flag.String("name", "", "The name of the module.")
var serverLoc = flag.String("server", "", "The server location to report the status back.")

type Config struct {
	XMLName       xml.Name `xml:"config"`
	SchedCfg      schedConn.SchedulerCfg
	Dbs           dbs.DbsConn `xml:"dbs"`
	Tables        []TableCfg  `xml:"tables>table"`
}

type TableCfg struct {
	XMLName  xml.Name	`xml:"table"`
	Name     string		`xml:"name,attr"`
	Size	 string		`xml:"size,attr"`
	Time	 string		`xml:"time,attr"`
	Interval int		`xml:"interval,attr"`
}

type watchTable struct {
	name		string
	schedName	string
	present		bool
	partTable	string
	token		string
	inputTime	int64
	targetDur	time.Duration
	targetSize	ByteSize
	actualDur	time.Duration
	actualSize	ByteSize
}

type extImport struct {
	tables []*watchTable
	maxNameLen int
}

type ByteSize int64

const (
	_			= iota
	KB ByteSize = 1<<(10*iota)
	MB
	GB
	TB
	PB
	EB
)

func ByteSizeFromString(inStr string) (size ByteSize) {
	sizeStr := strings.TrimSpace(strings.ToUpper(inStr))

	lenSizeStr := len(sizeStr)
	if sizeStr[lenSizeStr-1] == 'B' {
		largeness := sizeStr[lenSizeStr-2:lenSizeStr]
		sizeNum := strings.TrimSpace(sizeStr[0:lenSizeStr-2])
		sizeInt, err := strconv.ParseInt(sizeNum, 10, 64)
		size = ByteSize(sizeInt)
		if err != nil {
			log.Printf("ERROR: converting string %s to size failed\n", inStr)
		}
		switch {
			case largeness == "EB":
				size = size * EB
			case largeness == "PB":
				size = size * PB
			case largeness == "TB":
				size = size * TB
			case largeness == "GB":
				size = size * GB
			case largeness == "MB":
				size = size * MB
			case largeness == "KB":
				size = size * KB
		}
	} else {
		sizeInt, err := strconv.ParseInt(sizeStr, 10, 64)
		size = ByteSize(sizeInt)
		if err != nil {
			log.Printf("ERROR: converting string %s to size failed\n", inStr)
		}
	}
	return size
}

func (b ByteSize) String() string {
	switch {
		case b >= EB:
			return fmt.Sprintf("%.3f EB", float64(b)/float64(EB))
		case b >= PB:
			return fmt.Sprintf("%.3f PB", float64(b)/float64(PB))
		case b >= TB:
			return fmt.Sprintf("%.3f TB", float64(b)/float64(TB))
		case b >= GB:
			return fmt.Sprintf("%.3f GB", float64(b)/float64(GB))
		case b >= MB:
			return fmt.Sprintf("%.3f MB", float64(b)/float64(MB))
		case b >= KB:
			return fmt.Sprintf("%.3f KB", float64(b)/float64(KB))
		case b < 0:
			return "invalid"
	}
	return fmt.Sprintf("%.2fB", b)
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

	var e extImport
	e.tables = make([]*watchTable, len(cfg.Tables))
	for i, cfgTbl := range cfg.Tables {
		if len(cfgTbl.Name) > e.maxNameLen {
			e.maxNameLen = len(cfgTbl.Name)
		}
		size := ByteSizeFromString(cfgTbl.Size)
		if size < MB {
			log.Printf("ERROR: size of table %s < 1 MB", cfgTbl.Name)
		}
		log.Printf("size: %s in bytes: %d", cfgTbl.Size, size)
		newTbl := &watchTable{
			cfgTbl.Name,
			"ret_"+cfgTbl.Name,
			false,
			"",
			"",
			0,
			time.Duration(0 * time.Second),
			size,
			time.Duration(0 * time.Second),
			-1}

		e.tables[i] = newTbl
		go newTbl.Watch(cfgTbl.Interval)
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

	identStr := ""
	for i := 0; i < e.maxNameLen; i++ {
		identStr += " "
	}

	for _, tbl := range e.tables {
		identNum := e.maxNameLen-len(tbl.name)
		ident := ""
		if identNum > 0 {
			ident = identStr[0:identNum]
		}
		if tbl.present {
			status[tbl.name] = fmt.Sprintf("%ssize: %s (target:%s) dur: %s", ident, tbl.actualSize, tbl.targetSize, tbl.actualDur)
		} else {
			status[tbl.name] = "table not found."
		}
	}
	return status
}

func (e extImport) printStatus() {
	log.Println("Retention initialized with the following tables")
	for n, s := range e.Status() {
		log.Printf("%s: %s\n", n, s)
	}
}

func (t *watchTable) init(partTable string, size int) (bool) {
	t.partTable = partTable
	parts, err := dbs.GetPartSizeTimeSort(t.partTable)
	if err != nil {
		t.present = false
		return false
	}
	t.inputTime = int64(parts[len(parts)-1].MaxSerialTime)
	inp := utils.IOWindow{t.name, partTable, size, 0, true}
	outp := utils.IOWindow{"none", "", 0, 0, true}
	reply := schedConn.RegisterJobSeries(t.schedName, "", []utils.IOWindow{inp}, []utils.IOWindow{outp}, t.inputTime, 0)
	t.token = reply.RegToken
	return true
}

func (t *watchTable) deleteData(parts []dbs.DbsPartSize, totalSize ByteSize) {
	query := ""
	lenParts := len(parts)
	for i,p := range parts {
		if totalSize <= t.targetSize {
			break
		}
		if i == lenParts-1 {
			log.Printf("WARNING: tagetSize of table %s smaller than one partition (%s).", t.name, ByteSize(p.Size))
			break
		}
		totalSize -= ByteSize(p.Size)
		query += fmt.Sprintf("drop table if exists %s cascade; delete from %s where tablename='%s';\n", 
			p.Tablename, t.partTable, p.Tablename)
	}
	if len(query) > 0 {
		log.Printf("DeleQ: %s\n", query)
		dbs.Exec(query)
	}
}

func (t *watchTable) Watch(checkInterval int) {
	first := true
	var jobTime time.Time
	var reply remote.ExecReply
	var totalSize ByteSize
	var min_st int
	var max_st int
	for {
		if !t.present {
			tbl := dbs.GetTable(t.name)
			if len(tbl) == 0 {
				time.Sleep(time.Duration(30) * time.Second)
				continue
			} else {
				t.present =	t.init(tbl[0].PartTablename, checkInterval)
			}
		}
		parts, err := dbs.GetPartSizeTimeSort(t.partTable)
		if err != nil {
			t.present = false
			continue
		}
		totalSize = 0
		min_st = parts[0].MinSerialTime
		max_st = parts[len(parts)-1].MaxSerialTime
		for _, p := range parts {
			totalSize += ByteSize(p.Size)
		}
		if totalSize > t.targetSize {
			t.deleteData(parts, totalSize)
		}
		t.actualSize = ByteSize(totalSize)
		t.actualDur = time.Duration(time.Duration(max_st-min_st) * time.Second)
		if !first {
			_ = schedConn.SendJobDone(t.schedName, reply.JobToken, time.Since(jobTime), 0)
		}
		first = false

		t.inputTime += int64(checkInterval)
		_, reply = schedConn.SendAndWaitForJob(t.schedName, t.token, t.inputTime)
		jobTime = time.Now()
	}
}
