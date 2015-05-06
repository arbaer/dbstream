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
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
//	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/go-martini/martini"

	"../util"
	rep "../../../server/lib/reporting"
)

type Config struct {
	FileTimeConvMethod 	string 		`xml:"fileTimeConvMethod,attr"`
	PollInterval 		int 		`xml:"pollInterval,attr"`
	FileHandlerConfig 	FHConfig 	`xml:"fileHandler"`
}

type WebHandler struct {
	name 			string
	cfg 			Config
	imports 		[]*util.DBSImportConfig
	fileHandlers	[]FileHandler
	mutex 			*sync.Mutex
	runningImports  string
}

//var runningImports = "running_imports.xml"

type RunningImports struct {
	XMLName		xml.Name 				`xml:"runningImports"`
	Imports 	[]util.DBSImportConfig  `xml:"imports>DBSImportConfig"`
}

func NewWebHandler(name string, cfg Config) *WebHandler {
	imp := &WebHandler {
		name,
		cfg,
		make([]*util.DBSImportConfig, 0),
		make([]FileHandler, 0),
		new(sync.Mutex),
		cfg.FileHandlerConfig.OutDir + "/running_imports.xml",
	}	
	var imports RunningImports
	runningFile, err := os.Open(imp.runningImports)
	if os.IsNotExist(err) {
		log.Print("No running import file found starting with 0 imports.")
		return imp
	} else if err != nil {
		log.Fatalf("ERROR: running file Error: %v\n", err)
	}

	decode := xml.NewDecoder(runningFile)
	err = decode.Decode(&imports)
	if err != nil {
		log.Printf("State Decode Error: %v\n", err)
	}
	runningFile.Close()

	for _,cfg := range imports.Imports {
		log.Printf("Restarting import: %s", cfg.StreamName)
		imp.StartImport(cfg)
	}

	return imp
}


func  getFileList(url, streamName string) (files []string) {

	for {
		resp, err :=  http.Get("http://" + url + "/ListDir/" + streamName)
		if err != nil {
			log.Printf("ERROR probe not available: %+v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("ERROR: read list dir failed: %v", err)
		}

		err = json.Unmarshal(body, &files)
		if err != nil {
			log.Printf("ERROR: json unmarshal failed. %v\nData:%s", err, )
		}
		return files
	}
}

func (i *WebHandler) persistImports() {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	running := RunningImports{}
	for _, impCfg := range i.imports {
		running.Imports = append(running.Imports, *impCfg)
	}

	runningFile, err := os.Create(i.runningImports)
	if err != nil {
		log.Printf("ERROR: %v", err)
	}
	enc := xml.NewEncoder(runningFile)

	//this also calls flush
	if err := enc.Encode(running); err != nil {
		log.Fatalf("ERROR: %v\n", err)
	}
	runningFile.Close()
}

func (i *WebHandler) RunImport(cfg *util.DBSImportConfig) {
	log.Printf("Running Import")
	
	fq := make(chan FileTime)
	fh := GetFileHandler(i.cfg.FileHandlerConfig.Type)

	fh.Configure(i.cfg.FileHandlerConfig, *cfg, fq)
	curTime := fh.GetLastTimestamp()

	if curTime < cfg.StartTime {
		curTime = cfg.StartTime
	}

	go fh.HandleImport()

	i.persistImports()

	i.mutex.Lock()
	i.fileHandlers = append(i.fileHandlers, fh)
	i.mutex.Unlock()

	for {
		var fileTimes []FileTime

		files := getFileList(cfg.CallBackUrl, cfg.StreamName)

		for _,file := range files {
			ft := GetFileTime(i.cfg.FileTimeConvMethod, file)

			if ft.Timestamp > curTime {
				fileTimes = append(fileTimes, ft)
				curTime = ft.Timestamp
			}
		}
		for _, ft := range fileTimes {
			fmt.Printf("Downloading: \"%v\"...\n", ft)

			fq <- ft	 
		}

		time.Sleep(time.Duration(i.cfg.PollInterval) * time.Second)
	}
}

func (i *WebHandler) ReportStatus() {
	for {
		select {
		case <-time.After(1 * time.Second):
			status := make(map[string]string)

			i.mutex.Lock()
			for id, fh := range i.fileHandlers {
				status[i.imports[id].StreamName] = fh.GetStatus()
			}
			i.mutex.Unlock()

			rep.SendReport(i.name, status)
		}
	}
}

/*
	TODO: add security check for adding to many imports.
*/
func (i *WebHandler) StartImport(conf util.DBSImportConfig) (newId int) {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	//check if import already exists
	for id, imp := range i.imports {
		if imp.StreamName == conf.StreamName {
			log.Printf("Import: %s already exists, rejecting.", conf)
			return id
		}
	}

	newId = len(i.imports)

	i.imports = append(i.imports, &conf)

	go i.RunImport(&conf)

	return newId
}

func (i *WebHandler) GetImport(id int) (*util.DBSImportConfig, error) {
	// Check if we have a valid id.
	if id < 0 || id >= len(i.imports) || i.imports[id] == nil {
		return nil, fmt.Errorf("Import ID not found.")
	}
	return i.imports[id], nil
}

func (i *WebHandler) GetAllImports() map[string]util.DBSImportConfig {
	imports := make(map[string]util.DBSImportConfig, 0)
	for id, imp := range i.imports {
		if imp != nil {
			imports[fmt.Sprintf("%d", id)] = *imp
		}
	}
	return imports
}

/*
	RESTFull impl
*/

func (i *WebHandler) GetPath() string {
	return "/DBSImport"
}

func (i *WebHandler) RPost(params martini.Params, req *http.Request) (int, string) {
	defer req.Body.Close()

	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return http.StatusInternalServerError, "internal error"
	}

	if len(params) != 0 {
		return http.StatusMethodNotAllowed, "method not allowed"
	}

	var conf util.DBSImportConfig
	err = json.Unmarshal(requestBody, &conf)
	if err != nil {
		return http.StatusBadRequest, "invalid JSON data"
	}

	i.StartImport(conf)

	return http.StatusOK, "new entry created"
}


func (i *WebHandler) RDelete(params martini.Params) (int, string) {
	return http.StatusInternalServerError, "internal error"
}



func (i *WebHandler) RGet(params martini.Params) (int, string) {

	if len(params) == 0 {
		log.Printf("%v", i.GetAllImports())
		encodedEntries, err := json.Marshal(i.GetAllImports())
		if err != nil {
			log.Printf("%s", err)
			return http.StatusInternalServerError, "internal error"
		}
		log.Printf("%s", err)
		return http.StatusOK, string(encodedEntries)
	}

	id, err := strconv.Atoi(params["id"])
	if err != nil {
		log.Printf("%s", err)
		return http.StatusBadRequest, "invalid entry id"
	}
	// Get entry identified by id.
	entry, err := i.GetImport(id)
	if err != nil {
		log.Printf("%s", err)
		return http.StatusNotFound, "entry not found"
	}
	// Encode entry in JSON.
	encodedEntry, err := json.Marshal(entry)
	if err != nil {
		// .
		log.Printf("%s", err)
		return http.StatusInternalServerError, "internal error - Failed encoding entry"
	}
	// Return encoded entry.
	log.Printf("%s", err)
	return http.StatusOK, string(encodedEntry)	
}
