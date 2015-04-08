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
	"fmt"
	"flag"
	"log"

	"github.com/go-martini/martini"

	"./dbs_import"
	rep "../../server/lib/reporting"
	"./util"
)

var configFileName = flag.String("config", "math_repo.xml", "The configuration file used in the server. If not present, the file 'serverConf.xml' will be used.")
var verbose = flag.Bool("verbose", false, "Enables output of line numbers in the log.")
var serverLoc = flag.String("server", "", "The server location to report the status back. (Needed for hydra connection)")
var name = flag.String("name", "", "The name of the module. (Needed for hydra connection)")

type Config struct {
	XMLName		xml.Name			`xml:"config"`
	Ip 			string				`xml:"ip,attr"`
	Port 		string				`xml:"port,attr"`
	ImportCfg	dbs_import.Config 	`xml:"import"`
}

type RepoServer struct {
	cfg Config
	test string
}

func main() {
	flag.Parse()
	if *verbose {
		log.SetFlags(log.Llongfile)
	}

	fmt.Printf("Starting math_repo...\n")

	if flag.Lookup("config").Value.String() == flag.Lookup("config").DefValue {
		log.Printf("Using default config file %v\n", flag.Lookup("config").DefValue)
	}

	var cfg Config
	util.ReadConfig(*configFileName, &cfg)

	var server RepoServer
	server.cfg = cfg

	cm := martini.Classic()
	webHandler := dbs_import.NewWebHandler(*name, server.cfg.ImportCfg)

	util.RegisterRESTFunction(webHandler, cm)

	//enable hydra reporting if configured
	if len(*serverLoc) > 0 {
		log.Printf("Running with hydra connection.")
		err := rep.Configure(*serverLoc)
		if err != nil {
			log.Fatalf("ERROR: configure reporting error: %v\n", err)
		}
		go webHandler.ReportStatus()
	} else {
		log.Printf("Running in standalone mode.")
	}	

	cm.RunOnAddr(server.cfg.Ip + ":" + server.cfg.Port)
}
