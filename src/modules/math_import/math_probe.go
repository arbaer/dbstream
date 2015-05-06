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

import(
	"encoding/xml"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-martini/martini"

	"./util"

)

var configFileName = flag.String("config", "math_probe.xml", "The configuration file used in the server. If not present, the file 'serverConf.xml' will be used.")
var repoUrl = flag.String("repoUrl", "localhost:3000", "The URL under which to contact the Repository, e.g. localhost:3000")
var startTimeFlag = flag.String("startTime", "0", "The point in time the import should be started, e.g. 2006-01-02T15:04:05Z07:00")

type Config struct {
	XMLName		xml.Name	`xml:"config"`
	Ip 			string		`xml:"ip,attr"`
	Port 		string		`xml:"port,attr"`
	Directory	string 		`xml:"directory,attr"`
	StreamName 	string 		`xml:"streamName,attr"`
	StreamType 	string 		`xml:"streamType,attr"`
}

type Server struct {
	cfg Config
}

func (s *Server) ListDir(params martini.Params) (int, string) {
	reqStream := params["name"]

	if reqStream != s.cfg.StreamName {
		return http.StatusMethodNotAllowed, "Queried Stream not available."
	} else {
		files := make([]string, 0)

		filepath.Walk(s.cfg.Directory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Fatalf("ERROR: %s", err)
			}
			if !info.IsDir() {
				files = append(files, path)			
				log.Printf(path) 
			}			
			return nil
		})

		filesJson, err := json.Marshal(files)
		if err != nil {
			log.Print(err)
		}

		return http.StatusOK, string(filesJson)
	}
}

func main() {
	flag.Parse()

	var err error
	startTime := time.Now()
	if startTimeFlag != nil {
		startTime, err = time.Parse("2006-01-02T15:04:05Z07:00", *startTimeFlag)
		if err != nil {
			log.Fatalf("ERROR: given time format wrong:%s", err)
		}
	} 
	startTimeUnix := startTime.Unix()

	log.Printf("Starting import at: %s", startTime)
	importUrl := *repoUrl + "/DBSImport"

	var cfg Config
	util.ReadConfig(*configFileName, &cfg)

	var server Server
	server.cfg = cfg

	log.Printf("%v\n", cfg)

	cm := martini.Classic()

	fmt.Printf("Directory: %s\n", cfg.Directory)

	dirSplit := strings.Split(cfg.Directory, "/")

	log.Printf("%v", dirSplit)

	cm.Use(martini.Static(cfg.Directory, martini.StaticOptions{ Prefix:cfg.Directory }))

	cm.Get("/ListDir/:name", server.ListDir)

	go cm.RunOnAddr(cfg.Ip + ":" + cfg.Port)

	impCfg := util.DBSImportConfig {
		CallBackUrl: cfg.Ip + ":" + cfg.Port,
		StreamName: cfg.StreamName, 
		StreamType: cfg.StreamType,
		StartTime: startTimeUnix,
//		ImportDir: "PUL",
	}

	encodedCfg, err := json.Marshal(impCfg)
	if err != nil {
		log.Print(err)
	}

	_, err = http.Post("http://" + importUrl, "application/json", strings.NewReader(string(encodedCfg)))
	if err != nil {
		log.Printf("ERROR: %s", err)
	}

	for ;true; {
	 	time.Sleep(1 * time.Second)
	}
}
