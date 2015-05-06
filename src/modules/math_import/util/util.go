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
package util

import (
	"encoding/xml"
	"log"
	"net/http"
	"os"

	"github.com/go-martini/martini"
)

type DBSImportConfig struct {
	XMLName		xml.Name 	`xml:"DBSImportConfig"`
	CallBackUrl	string 		`xml:"callBackUrl,attr"`
	StreamName	string		`xml:"streamName,attr"`
	StreamType	string		`xml:"streamType,attr"`
	StartTime	int64 
//	ImportDir	string		`xml:"importDir,attr"`
}

type RESTFunction interface {
	GetPath() string
	RDelete(params martini.Params) (int, string)
	RGet(params martini.Params) (int, string)
	RPost(params martini.Params, req *http.Request) (int, string)
}

func RegisterRESTFunction(f RESTFunction, cm *martini.ClassicMartini) {
	path := f.GetPath()

	cm.Get(path, f.RGet)
	cm.Get(path+"/:id", f.RGet)

	cm.Post(path, f.RPost)
	cm.Post(path+"/:id", f.RPost)

	cm.Delete(path, f.RDelete)
	cm.Delete(path+"/:id", f.RDelete)
}

func ReadConfig(configFileName string, cfg interface{}) () {
	//decode the xml config file
	cfgFile, err := os.Open(configFileName)
	if err != nil {
		log.Fatalf("ERROR: Config file Error: %v\n", err)
	}
	decode := xml.NewDecoder(cfgFile)
	err = decode.Decode(cfg)
	if err != nil {
		log.Printf("Configuration Error: %v\n", err)
	}
	cfgFile.Close()
	return
}
