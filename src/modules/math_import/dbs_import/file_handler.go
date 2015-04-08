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
	"../util"
	"encoding/xml"
	"log"
	"net/http"
	"io"
	"os"
	"path"
	"path/filepath"
)

type FHConfig struct {
	Type 		string	`xml:"type,attr"`
	OutDir 		string	`xml:"outDir,attr"`
	InnerXml 	[]byte 	`xml:",innerxml"`	
}

/*
* FileHandler
*
* Interface to implement remote file handlers. The simplest implementation is the CopyFileHandler given in this file.
*/

type FileHandler interface {
	Configure(cfg FHConfig, importCfg util.DBSImportConfig, fileQueue chan FileTime)
	GetLastTimestamp() int64
	HandleImport()
	GetStatus() string
}

func GetFileHandler(fhType string) (fh FileHandler) {
	if fhType == "copy" {
		return &CopyFileHandler{} //{cfg, make(chan string, 100)}
	} else if fhType == "dbs" {
		return &DBSImportHandler{}
	}
	return nil
}

/*
* CopyFileHandler implementation
*/
type CopyFHState struct {
	XMLName   				xml.Name 	`xml:"state"`
	LastFinishedTimestamp	int64		`xml:"lastFinishedTimestamp,attr"`
}

type CopyFileHandler struct {
	isConfigured 	bool
	cfg 			FHConfig
	importCfg 		util.DBSImportConfig
	fileQueue 		chan FileTime
	copyFHStateFile string
	currentTime 	int64	
}

func (f *CopyFileHandler) Configure(cfg FHConfig, importCfg util.DBSImportConfig, fileQueue chan FileTime) {
	f.cfg = cfg
	f.importCfg = importCfg
	f.fileQueue = fileQueue
	f.isConfigured = true
	f.copyFHStateFile = "copy_stat_" + importCfg.StreamName + ".xml"
}

func (f *CopyFileHandler) GetLastTimestamp() int64 {
	if !f.isConfigured {
		log.Fatal("CopyFileHandler not configured, please call Configure first.")
	}

	stateFilename := f.cfg.OutDir + "/" + f.copyFHStateFile

	var state CopyFHState
	statFile, err := os.Open(stateFilename)
	if os.IsNotExist(err) {
		log.Print("No state file found starting import from the beginning.")
		return 0
	} else if err != nil {
		log.Fatalf("ERROR: State file Error: %v\n", err)
	}

	decode := xml.NewDecoder(statFile)
	err = decode.Decode(&state)
	if err != nil {
		log.Printf("State Decode Error: %v\n", err)
	}
	statFile.Close()

	return state.LastFinishedTimestamp
}

func (f *CopyFileHandler) HandleImport() {
	if !f.isConfigured {
		log.Fatal("CopyFileHandler not configured, please call Configure first.")
	}

	for file := range f.fileQueue {
		log.Printf("Next File: %v\n", file)
		f.currentTime = file.Timestamp

		resp, err :=  http.Get("http://" + f.importCfg.CallBackUrl + "/" + file.Filename)
		if err != nil {
			log.Printf("ERROR: %v", err)
		}

		fileDir := path.Join(f.cfg.OutDir, f.importCfg.StreamName, file.Filename)
		os.MkdirAll(filepath.Dir(fileDir), 0777)

		dest, err := os.Create(fileDir)
		if err != nil {
			log.Printf("ERROR: %v", err)
		}
		io.Copy(dest, resp.Body)

		resp.Body.Close()

		// write state to disk
		state := CopyFHState{LastFinishedTimestamp: file.Timestamp}		

		statFile, err := os.Create(f.cfg.OutDir + "/" + f.copyFHStateFile)
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
}

func (f *CopyFileHandler) GetStatus() string {
	return fmt.Sprintf("CurrentTime: %s", f.currentTime)
}
