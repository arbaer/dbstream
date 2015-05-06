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
package dbs_import

import (
	"fmt"
	"log"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"time"
	schedConn "../../../modules/scheduler/conn"
	"../../../server/utils"
	"../../../server/lib/dbs"
)

/*
* DBSTableWriterConfig
*
* Handles the import of equally sized (in terms of time) files (e.g. 10 minute long) into DBStream.
*/

type DBSTableWriterConfig struct {
	XMLName   		xml.Name 	`xml:"dbsTable"`
	TableInterval 	int64 		`xml:"tableInterval,attr"`
}

type DBSTableWriter struct {
	cfg 			dbs.DbsConn
	schedCfg      	schedConn.SchedulerCfg
	inFileTime 		int64
	outFileTime 	int64

	streamName 		string
	streamType 		string
	partSchema 		string
	outputWindow 	int64
	fileChan 		chan FileTime

	startTime 		int64
	CurrentTime 	int64
	partTablename 	string
	stateFilename 	string
}

func NewDBSTableWriter(cfg dbs.DbsConn, schedCfg schedConn.SchedulerCfg, inFileTime, outFileTime int64, stateFilename string, streamName, streamType, partSchema string, 
						outputWindow int64, fileChan chan FileTime) *DBSTableWriter {

	var tw DBSTableWriter

	tw.cfg = cfg
	tw.schedCfg = schedCfg
	tw.inFileTime = inFileTime
	tw.outFileTime = outFileTime

	tw.stateFilename = stateFilename
	tw.streamName = strings.ToLower(streamName)
	tw.streamType =	streamType
	tw.partSchema = partSchema
	tw.outputWindow = outputWindow
	tw.fileChan = fileChan

	err := dbs.Configure(tw.cfg)
	if err != nil {
		log.Fatalf("ERROR: configure TicketDB error: %v\n", err)
	}

	err = schedConn.Configure(tw.schedCfg)
	if err != nil {
		log.Fatalf("ERROR: configure scheduler error: %v\n", err)
	}

	tw.initJobState()
	return &tw
}

func (w *DBSTableWriter) DoImport() {
	defer func() {
        if err := recover(); err != nil {
            log.Println("work failed:", err)
        }
    }()

	outp := utils.IOWindow{w.streamName, w.partTablename, 0, 0, true}
	reply := schedConn.RegisterJobSeries(w.streamName, "", nil, []utils.IOWindow{outp}, w.outFileTime, 0)
	token := reply.RegToken
	lastImport := time.Now()

	//log.Printf("XXXXXXXXXXXXXX waiting for files")
	for file := range w.fileChan {
		//ask scheduler for execution permission

		//log.Printf("XXXXXXXXXXXXXX sending job")
		_, execReply := schedConn.SendAndWaitForJob(w.streamName, token, file.Timestamp)
		//log.Printf("XXXXXXXXXXXXXX job exec")
		w.CurrentTime = file.Timestamp
		lastImport = time.Now()
		//create new partition
		log.Printf("NextFile: %s", file.Filename)
		part, createQ := w.getCreateNewPartitionQ(file.Timestamp, w.outputWindow)
		absPath, err := filepath.Abs(file.Filename)
		if err != nil {
			panic(err)
		}
		//import file
		dbs.Execf(createQ + "copy %s from '%s' (NULL '-'); analyze %s;", part, absPath, part)

		//insert row into partition table
		partTblQ := fmt.Sprintf("insert into %s values (%d, %d, $$%s$$, %d, $$%s$$, $$%s$$)", 
			w.partTablename, file.Timestamp, w.outputWindow, 
			dbs.GetViewHostname(), dbs.GetViewPort(), dbs.GetViewDBname(), part)
		dbs.Exec(partTblQ)

		//delete imported file on disk
		err = os.Remove(file.Filename)		
		if err != nil {
			panic(err)
		}

		w.WriteState(file.Timestamp)

		//inform DBStream scheduler about new data
		_ = schedConn.SendJobDone(w.streamName, execReply.JobToken, time.Since(lastImport), 0)

	}
}

func (w *DBSTableWriter) getCreateNewPartitionQ(now, step int64) (curPart, createQ string) {
	curPart = strings.ToLower(fmt.Sprintf("%s.%s_%d", w.partSchema, w.streamName, now))
	dropQ := fmt.Sprintf("drop table if exists %v cascade", curPart)
	log.Printf("dropQ: %s\n", dropQ)
	dbs.Exec(dropQ)

	createQ = fmt.Sprintf("create table %v (check (serial_time between %v and %v)) inherits (%v)", curPart, now, now+step-1, w.streamName)
	if len(w.partSchema) > 0 {
		createQ += fmt.Sprintf(" tablespace %s", w.partSchema)
	}
	createQ += ";"
	log.Printf("createQ: %s\n", createQ)
	
	return curPart, createQ
}

func (w *DBSTableWriter) initJobState() {
	partTblname, _/*curPartTblname*/, lastImport := dbs.GetOrCreateJobState(strings.ToLower(w.streamName))
	likeStreamType := fmt.Sprintf("like %s", w.streamType)
	w.partTablename = partTblname
	if lastImport <= 0 {
		dbs.CreateViewParent(strings.ToLower(w.streamName), likeStreamType)
		if w.startTime > 0 {
			w.CurrentTime = w.startTime - w.startTime % int64(w.outputWindow)
		} else {
			//if there is no start time given use the current time as start time
			//w.CurrentTime = time.Now().Unix() - time.Now().Unix() % int64(w.outputWindow)
			w.CurrentTime = 0
		}
	} else {
		//check if the window size changed
		if lastImport % int64(w.outputWindow) == 0 {
			w.CurrentTime = lastImport + int64(w.outputWindow)
		} else {
			log.Fatalf("Window size of input %s changed, please use new table.", w.streamName)
		}
	}
}

func (w *DBSTableWriter) WriteState(lastOutputTime int64) {
	state := DBSFHState{
		LastInputFileTime: w.inFileTime -1,
		LastOutputFileTime: lastOutputTime + w.outputWindow,
	}		

	log.Printf("Writing State: %v", state)

	statFile, err := os.Create(w.stateFilename)
	if err != nil {
		log.Printf("ERROR: %v", err)
	}
	enc := xml.NewEncoder(statFile)

	//this also calls flush
	if err := enc.Encode(state); err != nil {
		log.Fatalf("ERROR: %v\n", err)
	}
	statFile.Close()
}
