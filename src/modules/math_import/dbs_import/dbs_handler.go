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
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"	
	"path/filepath"
	"time"

	//"compress/gzip" //original go gzip library
	"code.google.com/p/vitess/go/cgzip" //faster gzip library

	"../../../server/lib/dbs"
	schedConn "../../scheduler/conn"
	"../util"
)

var delimiter = "\t"

type DBSFHState struct {
	XMLName   				xml.Name 	`xml:"state"`
	LastInputFileTime 		int64		`xml:"lastInputFileTime,attr"`
	LastOutputFileTime 		int64		`xml:"lastOutputFileTime,attr"` 
}

type DBSFHConfig struct {
	XMLName 		xml.Name 	`xml:dbsHandler`
	PartSchema 		string 		`xml:"partSchema,attr"`
	InputWindow		int64		`xml:"inputWindow,attr"`
	OutputWindow	int64		`xml:"outputWindow,attr"`
	Dbs        		dbs.DbsConn	`xml:"dbs"`
	SchedCfg      schedConn.SchedulerCfg
}

/*
* DBSImportHandler
*
* Handles the import of files from probes into DBStream.
* First,the DBSFileWriter is used to write data into files and then 
* the DBSTableWriterConfig copies the data into newly created DBStream continuous table partitions.
*/

type DBSImportHandler struct {
	isConfigured 	bool
	dbsCfg 			DBSFHConfig
	cfg 			FHConfig
	importCfg 		util.DBSImportConfig
	fileQueue 		chan FileTime
	remoteFileTime 	int64
	lastSerialTime 	int64	
	writer 			*DBSFileWriter
	tableWriter 	*DBSTableWriter
	stateFilename 	string
	state 			DBSFHState
	inConv InputConverter
}

func (f *DBSImportHandler) Configure(cfg FHConfig, importCfg util.DBSImportConfig, fileQueue chan FileTime) {
	f.cfg = cfg
	f.importCfg = importCfg
	f.fileQueue = fileQueue
	f.isConfigured = true

	f.inConv = GetInputConverter(importCfg.StreamType)

	xml.Unmarshal(cfg.InnerXml, &f.dbsCfg)

	if _, err := os.Stat(f.cfg.OutDir); os.IsNotExist(err) {
		// path/to/whatever does not exist
		os.Mkdir(f.cfg.OutDir, 0777)
	}
	

	//recover state
	f.stateFilename = f.cfg.OutDir + "/dbs_stat_" + importCfg.StreamName + ".xml"

	var state DBSFHState
	statFile, err := os.Open(f.stateFilename)
	if os.IsNotExist(err) {
		log.Print("No state file found starting import from the beginning.")

		statFile, err = os.Create(f.stateFilename)
		if err != nil {
			log.Fatalf("ERROR: Creating state file: %v\n", err)
		}

	} else if err != nil {
		log.Fatalf("ERROR: Opening state file: %v\n", err)
	}

	decode := xml.NewDecoder(statFile)
	err = decode.Decode(&state)
	if err != nil {
		log.Printf("State Decode Error: %v\n", err)
	}
	statFile.Close()

	f.state = state
}

func (f *DBSImportHandler) GetLastTimestamp() int64 {
	if !f.isConfigured {
		log.Fatal("DBSImportHandler not configured, please call Configure first.")
	}

	return f.state.LastInputFileTime
}

func (f *DBSImportHandler) HandleImport() {
	if !f.isConfigured {
		log.Fatal("DBSImportHandler not configured, please call Configure first.")
	}

	log.Printf(fmt.Sprintf("InputWindow: %d", f.dbsCfg.InputWindow))

	for file := range f.fileQueue {

		reader := f.openRemoteFile(file)

		f.convertAndWrite(reader, file.Timestamp)

		reader.Close()
	}
}

func convTime(t int64) string {
	return time.Unix(t, 0).Format("06-01-02 15:04:05")
}

func (f *DBSImportHandler) GetStatus() (status string) {
	if f.writer != nil {
		status = fmt.Sprintf("remote:%s handler:%s fw:%s tw:%s", 
		convTime(f.remoteFileTime), convTime(f.lastSerialTime), convTime(f.writer.LastFileTime), convTime(f.tableWriter.CurrentTime))
	} else {
		status = "waiting for probe to become available."
	}
	return status
}

/* private functions */
func (f *DBSImportHandler) convertAndWrite(source io.ReadCloser, remoteFileTime int64) {
	bufReader := bufio.NewReaderSize(source, 2<<24)
	f.remoteFileTime = remoteFileTime

	in := make(chan []string, 1000)
	done := make(chan bool)
	go f.handleLines(in, done, remoteFileTime)	
	log.Printf("Starting import.")
	for {
		line, err := bufReader.ReadString('\n')
		if err == io.EOF {
			break //file ended
		} else if err != nil {
			log.Fatalf("ERROR: %s\nWith line: %s", err, line)
		}
		if line[0] == '#' {
			continue //comments are skipped
		}

		in <- f.inConv.Line2Array(line)
	}	
	close(in)
	//wait for other thread to finish processing
	<-done
	if f.writer != nil {
		err := f.writer.Flush()
		if err != nil {
			log.Printf("ERROR: %s", err)
		}
	}	
}

func (f *DBSImportHandler) handleLines(lines <-chan []string, done chan bool, remoteFileTime int64) {
	sline, more := <- lines

	fileChan := make(chan FileTime, 0)

	if f.tableWriter == nil {
		f.tableWriter = NewDBSTableWriter(f.dbsCfg.Dbs,
			f.dbsCfg.SchedCfg,
			remoteFileTime,
			f.state.LastOutputFileTime,
			f.stateFilename,
			f.importCfg.StreamName,
			f.importCfg.StreamType,
			f.dbsCfg.PartSchema, 
			f.dbsCfg.OutputWindow, 
			fileChan)
		go f.tableWriter.DoImport()
	}
	f.tableWriter.inFileTime = remoteFileTime

	if f.writer == nil {		
		f.writer = NewDBSFileWriter(
			f.inConv.GetSerialTime(sline, f.lastSerialTime), 
			f.dbsCfg.InputWindow, 
			f.cfg.OutDir,
			f.importCfg.StreamName, 
			fileChan)
	}

	for more  {
		if !f.inConv.CheckLine(sline) {
			log.Fatalf("ERROR: line failed format checks.")
		}

		curTime := f.inConv.GetSerialTime(sline, f.lastSerialTime)

		if curTime >= f.state.LastOutputFileTime {
			f.writer.WriteString(curTime, f.inConv.ConvertLine(sline, delimiter))
			f.lastSerialTime = curTime
		}
		sline, more = <- lines
	}
	done <- true
}

func (f *DBSImportHandler) openRemoteFile(file FileTime) io.ReadCloser {
	log.Printf("Next File: %v\n", file)

	resp, err :=  http.Get("http://" + f.importCfg.CallBackUrl + "/" + file.Filename)
	if err != nil {
		log.Printf("ERROR: %v", err)
	}

	var reader io.ReadCloser
	//Use gzip decompression in case files have the extension .gz
	if filepath.Ext(file.Filename) == ".gz" {
		reader, err = cgzip.NewReader(resp.Body)
		if err != nil {
			log.Panicf("ERROR: %s", err)
		}
	} else {
		reader = resp.Body
	}

	return reader
}
