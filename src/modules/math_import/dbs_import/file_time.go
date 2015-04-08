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
	"time"
	"log"
	"path"
)

/*
* FileTime
*
* Simple struct and parsing methods to infer the start timestamp of the data in the given file based on the path and filename.
*/
type FileTime struct {
	Timestamp int64
	Filename string
}

type ByTime []FileTime

func (a ByTime) Len() int           { return len(a) }
func (a ByTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTime) Less(i, j int) bool { return a[i].Timestamp < a[j].Timestamp }

/*
	This function is used to extract a unix timestamp from the filename of the file
*/
func GetFileTime(format, filename string) (ft FileTime) { //TODO: do this check only once at startup
	if format == "tstat" {
		return getFileTime_tstat(filename)
	} else {
		log.Fatalf("File Conversion Method: %s not implemented.", format)
	}
	return ft
}

/* Tstat file format implementation */
const tstat_shortForm = "2006_01_02_15_04.out"
func getFileTime_tstat(filename string) (ft FileTime) {
	dir, _ := path.Split(filename)
	_, lastDirName := path.Split(dir[:len(dir)-1])

	t, err := time.ParseInLocation(tstat_shortForm, lastDirName, time.Now().Location())
	if err != nil {	
		log.Panic(err)
	}
	ft.Timestamp = t.Unix()
	ft.Filename = filename

	return ft
}
/* /Tstat file format implementation */
