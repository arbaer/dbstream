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
	"fmt"
	"io"
	"log"
	"os"	
	"path"
	"path/filepath"
	"time"
	"strconv"
)

/* 
* DBSFileWriter
*  
* Handles the writing of files of a fixed size receiving input lines
*/

type DBSFileWriter struct {		
	file			io.WriteCloser
	writer 			*bufio.Writer
	fileChan 	chan FileTime
	LastFileTime 	int64	
	interval 		int64
	dirName			string
	streamName 		string
}

func NewDBSFileWriter(startTimestamp, interval int64, dirname, streamName string, fileChan chan FileTime) *DBSFileWriter {
	log.Printf("startTimestamp:%d", startTimestamp)
	return &DBSFileWriter{ nil, 
		nil, 
		fileChan,
		startTimestamp, 
		interval,
		dirname,
		streamName,
	}
}

func (d *DBSFileWriter) getFileName(fileTime int64) string {
	return path.Join(
		d.dirName,
		fmt.Sprintf("%s_%d_%s.txt", d.streamName, fileTime, time.Unix(fileTime, 0).Format("2006_01_02_15-04-05")))
}

func (d *DBSFileWriter) WriteString(timestamp int64, s string) (int, error) {
	if d.file == nil || timestamp - d.LastFileTime >= d.interval {

		if d.file != nil {
			d.writer.Flush()		
			d.file.Close()

			finishedFile := FileTime{d.LastFileTime, d.getFileName(d.LastFileTime)}
			d.fileChan <- finishedFile
		}
		d.LastFileTime = timestamp -  (timestamp % d.interval)
		log.Printf("Creating new writer. timestamp:%d, last:%d, interval:%d\n", timestamp, d.LastFileTime, d.interval)
		
		fileDir := d.getFileName(d.LastFileTime)
		os.MkdirAll(filepath.Dir(fileDir), 0777)

		dest, err := os.Create(fileDir)
		if err != nil {
			log.Printf("ERROR: %v", err)
		}
		d.file = dest
		d.writer = bufio.NewWriter(d.file)
	}

	n, err := d.writer.WriteString(strconv.FormatInt(timestamp, 10))
	if err != nil {return n, err}
	n, err = d.writer.WriteString(delimiter)
	if err != nil {return n, err}

	return d.writer.WriteString(s)
}

func (b *DBSFileWriter) Flush() error {
	if b.writer != nil {
		return b.writer.Flush()
	} 
	return nil
	
}

func (b *DBSFileWriter) Close() error {
	return b.file.Close()
}
