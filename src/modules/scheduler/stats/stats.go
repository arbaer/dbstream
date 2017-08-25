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
package stats

import (
	"fmt"
	"log"
	"time"

	//"github.com/lxn/go-pgsql"
	"github.com/jackc/pgx"
	
	"../../../server/lib/dbs"
)

var col Collector

type Collector struct {
	conn 			*pgx.Conn
	insertStmtName 	string

	//params
	timestamp		time.Time
	jobSeries		string
	job				string
	inputs			string
	outputs			string
	inputTime		int64
	execTime		time.Duration
	affectedRows	int64
	additional		map[string]string
}

func Configure() (err error) {
	//insertQ := "insert into dbs.query_stats values (@timestamp, @jobSeries, @job, @inputs, @outputs, @inputTime, @execTime, @affectedRows, @additional);"
	insertQ := "insert into dbs.query_stats values ($1, $2, $3, $4, $5, $6, $7, $8, $9);"
	//col.timestampParam		= pgsql.NewParameter("@timestamp", pgsql.Timestamp)
	//col.jobSeriesParam		= pgsql.NewParameter("@jobSeries", pgsql.Text)
	//col.jobParam			= pgsql.NewParameter("@job", pgsql.Text)
	//col.inputsParam			= pgsql.NewParameter("@inputs", pgsql.Text)
	//col.outputsParam		= pgsql.NewParameter("@outputs", pgsql.Text)
	//col.inputTimeParam		= pgsql.NewParameter("@inputTime", pgsql.Bigint)
	//col.execTimeParam		= pgsql.NewParameter("@execTime", pgsql.Real)
	//col.affectedRowsParam	= pgsql.NewParameter("@affectedRows", pgsql.Bigint)
	//col.additionalParam		= pgsql.NewParameter("@additional", pgsql.Text)
	col.insertStmtName = "collectStats"
	col.conn, err = dbs.Prepare(col.insertStmtName, insertQ)//, col.timestampParam, col.jobSeriesParam, col.jobParam, col.inputsParam, col.outputsParam, col.inputTimeParam, col.execTimeParam, col.affectedRowsParam, col.additionalParam)
	if err != nil {
		log.Fatalf("ERROR: stats collector query prepare failed. %v\n", err)
	}
	return
}

func dur2secs(dur time.Duration) (secs float32) {
	secs = float32(dur.Hours()*3600)
	secs += float32(dur.Minutes()*60)
	secs += float32(dur.Seconds())
	secs += float32(dur.Nanoseconds())*float32(0.000000001)
	return secs
}

func Collect(timestamp time.Time, jobSeries, job, inputs, outputs string, inputTime int64, execTime time.Duration, affectedRows int64, additional map[string]string) {

	//col.timestampParam.SetValue(timestamp)
	//col.jobSeriesParam.SetValue(jobSeries)
	//col.jobParam.SetValue(job)
	//col.inputsParam.SetValue(inputs)
	//col.outputsParam.SetValue(outputs)
	//col.inputTimeParam.SetValue(inputTime)
	//col.execTimeParam.SetValue(float32(execTime.Seconds()))
	//col.affectedRowsParam.SetValue(affectedRows)
	additionalString := ""
	for k,v := range additional {
		if len(additionalString) > 0 {
			additionalString += ", "
		}
		additionalString += fmt.Sprintf("%s=>\"%s\"", k, v)
	}
//	col.additionalParam.SetValue(additionalString)
	_, err := col.conn.Exec(col.insertStmtName, timestamp, jobSeries, job, inputs, outputs, inputTime, execTime, affectedRows, additional)
	if err != nil {
		log.Fatalf("ERROR: can not write statistics to database. %v", err)
	}
}
