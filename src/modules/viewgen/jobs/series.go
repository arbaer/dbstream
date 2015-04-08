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
package jobs

import (
	"fmt"
	"log"
	"net/rpc"
	"strings"
	"time"

	schedConn "../../scheduler/conn"
	"../../scheduler/remote"
	"../../../server/lib/dbs"
	"../../../server/utils"
)

type JobDefinition struct {
	Description string `xml:"description,attr"`
	Inputs      string `xml:"inputs,attr"`
	Output      string `xml:"output,attr"`
	Priority    int    `xml:"priority,attr"`
	StartTime   int64  `xml:"startTime,attr"`
	Schema      string `xml:"schema,attr"`
	Index		string `xml:"index,attr"`
	Query       string `xml:"query"`
}

type JobSeries struct {
	Name        string
	Running		bool
	Description string
	PartSchema  string
	inputs      []utils.IOWindow
	output      utils.IOWindow
	priority    int
	startTime   int64
	schema      string
	query       string
	index		[]string

	client       *rpc.Client
	token        string
	jobStartTime time.Time
	currentTime  int64
	primaryWnd   *utils.IOWindow

	partTablename  string
	curPartTblname string
}

func csvToSlice(csv string) (out []string) {
	if strings.Index(csv, ",") > 0 {
		out = strings.Split(csv, ",")
		for i := 0; i < len(out); i++ {
			out[i] = strings.Trim(out[i], " \t")
		}
	} else {
		if len(csv) > 0 {
			out = make([]string, 1)
			out[0] = strings.Trim(csv, " \t")
		}
	}
	return out
}

func (s *JobSeries) initJobState() {
	partTblname, curPartTblname, lastImport := dbs.GetOrCreateJobState(s.output.Name)
	dbs.CreateViewParent(s.output.Name, s.schema)
	s.partTablename = partTblname
	s.curPartTblname = curPartTblname
	if lastImport <= 0 {
		if s.startTime > 0 {
			s.currentTime = s.startTime - s.startTime % int64(s.primaryWnd.Size)
		} else {
			//if there is no start time given use the current time as start time
			s.currentTime = time.Now().Unix() - time.Now().Unix() % int64(s.primaryWnd.Size)
		}
	} else {
		//check if the window size changed
		if lastImport % int64(s.primaryWnd.Size) == 0 {
			s.currentTime = lastImport + int64(s.primaryWnd.Size)
		} else {
			//the window size changed, we have to do something
			s.currentTime = lastImport - lastImport % int64(s.primaryWnd.Size)
			dbs.DeleteFromParttable(s.partTablename, s.currentTime, -1)
			s.curPartTblname = ""
		}
	}
}

func (s *JobSeries) Status() string {
	inputs := ""
	for i, in := range s.inputs {
		if i != 0 {
			inputs += ","
		}
		inputs = fmt.Sprintf("%s %s wndSize:%d delay:%d", inputs, in.Name, in.Size, in.Delay)
	}
	output := fmt.Sprintf("%s wndSize:%d delay:%d", s.output.Name, s.output.Size, s.output.Delay)
	return fmt.Sprintf("startTime:%d currentTime:%v(%d) priority:%d inputs:%s outputs:%s", s.startTime, time.Unix(s.currentTime, 0), s.currentTime, s.priority, inputs, output)
}

func (s *JobSeries) Register(jd JobDefinition, ip, port string) {
	// generate jobSeries from config
	s.output = utils.IOWindowFromString(jd.Output)
	s.Name = s.output.Name
	s.Description = jd.Description

	inputs := csvToSlice(jd.Inputs)
	s.inputs = make([]utils.IOWindow, len(inputs))
	for i, inp := range inputs {
		s.inputs[i] = utils.IOWindowFromString(inp)
	}
	//find primary window or die
	if len(s.inputs) == 0 {
		s.primaryWnd = &s.output
	} else if len(s.inputs) == 1 {
		s.primaryWnd = &s.inputs[0]
	} else {
		for i, inWnd := range s.inputs {
			if inWnd.Primary {
				if s.primaryWnd == nil {
					s.primaryWnd = &s.inputs[i]
				} else {
					log.Panicf("ERROR: More than one primary window defined for job %s, %s.", s.Name, s.Description)
				}
			}
		}
		if s.primaryWnd == nil || s.primaryWnd.Size == 0 {
			log.Panicf("ERROR: No primary window found for job %s, %s. Please define one window to be primary e.g. tablename (window 60 delay 60 primary).", s.Name, s.Description)
		}
	}
	if s.output.Size%s.primaryWnd.Size != 0 {
		log.Panicf("ERROR: The size of the output window (%s: %d) has to be a multiple of the size of the primary window (%s: %d) for job %s, %s.", s.output.Name, s.output.Size, s.primaryWnd.Name, s.primaryWnd.Size, s.Name, s.Description)
	}

	s.index = csvToSlice(jd.Index)
	s.priority = jd.Priority
	s.startTime = jd.StartTime
	s.schema = jd.Schema
	s.query = strings.Trim(jd.Query, "\r\n\t ")

	//check job series configuration
	if !(strings.HasPrefix(s.schema, "serial_time int4") ||
		strings.HasPrefix(s.schema, "serial_time integer")) {

		log.Panicf(`ERROR: Job %v's schema: %v  does not start with a timestamp column named serial_time.
				All view schemas have to start with a unix timestamp e.g.: serial_time int4`,
			s.Name, s.schema)
	}

	s.initJobState()
}

func (s *JobSeries) registerInputs() (success bool) {
	dbsTables := dbs.GetTables()

	found := false
	for i, inp := range s.inputs {
		found = false
		for _, tbl := range dbsTables {
			if inp.Name == tbl.Tablename {
				log.Printf("Input: %+v\n", tbl)
				inp.PartTablename = tbl.PartTablename
				found = true
				s.inputs[i] = inp
			}
		}
		if !found {
			log.Printf("Input table for input: %s not found.", inp.Name)
			return false
		}
	}
	return true
}

func (s *JobSeries) Run() {
	for !s.registerInputs() {
		log.Println("Waiting for input to become available.")
		time.Sleep(5 * time.Second)
	}

	reply := schedConn.RegisterJobSeries(s.output.Name, s.Description, s.inputs, []utils.IOWindow{s.output}, s.currentTime, s.output.Delay)
	s.token = reply.RegToken

	tableStep := int64(s.output.Size)
	//find the step size for the job series
	step := int64(s.primaryWnd.Size)
	//if no old partition was found create a new one
	if s.curPartTblname == "" {
		lastTblTime := s.currentTime - s.currentTime%tableStep-int64(s.output.Delay)
		s.curPartTblname = s.createNewPartition(lastTblTime, tableStep)
	} else {
		//delete data which is newer than one step
		delQ := fmt.Sprintf("lock table %s in share mode; delete from %s where serial_time >= %d", s.curPartTblname, s.curPartTblname, s.currentTime)
		log.Println("delQ: " + delQ)
		aff_rows := dbs.Execf(delQ)
		log.Printf("delQ: %d rows deleted\n", aff_rows)
	}
	s.Running = true
	for s.Running {
		t := s.currentTime

		if (t-int64(s.output.Delay))%tableStep == 0 {
			//analyze old table
			dbs.Execf("analyze %s", s.curPartTblname)
			//create indices on old table
			if len(s.index) > 0 {
				s.createIndexIfNotExists(s.curPartTblname)
			}
			//create next partition
			s.curPartTblname = s.createNewPartition(t, tableStep)
		}
		if s.curPartTblname != "" {
			job, token := s.sendAndWaitForJob(s.currentTime + step)
			log.Printf("Job: %v\n", job)
			s.jobStartTime = time.Now()
			var insertQ string
			var partQ string

			views, viewCreateQ := s.createInputViews(s.currentTime + step)
			viewDeleteQ := s.deleteInputViews(views)
			insertQ, partQ = s.buildInsertQueryViews(s.curPartTblname, s.currentTime, step, views)

			insertQ = viewCreateQ + insertQ + ";" + viewDeleteQ
			log.Printf("insertQ: %q", strings.Replace(strings.Replace(insertQ, "\n", "", -1), "\t", "", -1))
			affectedRows := dbs.Exec(insertQ)
			log.Printf("partQ: %q", partQ)
			_ = dbs.Exec(partQ)

			s.sendJobDone(job, token, affectedRows)
		}
		log.Printf("Current time: %d, step: %d, tableStep: %d\n", s.currentTime, step, tableStep)
		s.currentTime += int64(step)
	}
}

func getViewName(inWnd utils.IOWindow, inputTime int64) (tblName string) {
	return fmt.Sprintf("%s_%d_%d", inWnd.Name,
		inWnd.NextStartTs(inputTime), inWnd.NextEndTs(inputTime))
}

func createView(viewname string, inWnd utils.IOWindow, start_ts, end_ts int64) (query string) {
	query = fmt.Sprintf("create temp view %s as (", viewname)
	query +=  dbs.GetSliceQuery(inWnd.Name, inWnd.PartTablename, start_ts, end_ts) + ");"

	log.Printf("View creation query: %s", query)
	return query
}

func (s *JobSeries) createInputViews(inputTime int64) (views map[string]string, query string) {
	views = make(map[string]string)

	boundary := s.primaryWnd.NextEndTs(inputTime) + 1

	for _,inWnd := range s.inputs {
		viewname := getViewName(inWnd, inputTime)

		start_ts := boundary - int64(inWnd.Size + inWnd.Delay)
		end_ts := boundary - int64(inWnd.Delay) - 1
		log.Printf("createInputViews inputTime:%v start_ts:%v end_ts:%v", inputTime, start_ts, end_ts)
		query  += createView(viewname, inWnd, start_ts, end_ts)

		views[inWnd.String()] = viewname
	}
	return views, query
}

func (s *JobSeries) deleteInputViews(views map[string]string) (query string) {
	for _,viewname := range views {
		query += fmt.Sprintf("drop view %s;", viewname)

	}
	return query
}

func (s *JobSeries) createIndexIfNotExists(curPart string) {
	for _, idx := range s.index {
		indexQ := ""
		partNoSchema := curPart[strings.Index(curPart, ".")+1:]
		indexName := fmt.Sprintf("%s_%s_idx", partNoSchema, idx)
		if dbs.Execf("select * from pg_tables where tablename = '%s'", partNoSchema) > 0 {
			if dbs.Execf("select * from pg_indexes where indexname = '%s'", indexName) > 0 {
				//index already exists
				continue
			} else {
				indexQ = fmt.Sprintf("create index %s on %s (%s);", indexName, curPart, idx)
				log.Printf("indexQ: %s\n", indexQ)
				dbs.Exec(indexQ)
			}
		}
	}
	return
}

func (s *JobSeries) createNewPartition(now, step int64) (curPart string) {
	curPart = fmt.Sprintf("%s.%s_%d", s.PartSchema, s.output.Name, now)
	dropQ := fmt.Sprintf("drop table if exists %v cascade", curPart)
	log.Printf("dropQ: %s\n", dropQ)
	dbs.Exec(dropQ)

	createQ := fmt.Sprintf("create table %v (check (serial_time between %v and %v)) inherits (%v)", curPart, now, now+step-1, s.output.Name)
	if len(s.PartSchema) > 0 {
		createQ += fmt.Sprintf(" tablespace %s", s.PartSchema)
	}
	log.Printf("createQ: %s\n", createQ)
	dbs.Exec(createQ)
	return curPart
}

func (s *JobSeries) buildInsertQueryViews(curPartTbl string, now, step int64, inputTables map[string]string) (insertQ, partTblQ string) {
	endTs := now + step
	//replace occurences of __STARTTS and __ENDTS with the actual timestamps
	query := strings.Replace(s.query, "__STARTTS64", fmt.Sprintf("%d", uint64(now)<<32), -1)
	query = strings.Replace(query, "__ENDTS64", fmt.Sprintf("%d", uint64(endTs)<<32), -1)
	query = strings.Replace(query, "__STARTTS", fmt.Sprintf("%d", now), -1)
	query = strings.Replace(query, "__ENDTS", fmt.Sprintf("%d", endTs), -1)
	for i, inWnd := range s.inputs {
		wnd := fmt.Sprintf("__IN%d", i)
		query = strings.Replace(query, wnd, inputTables[inWnd.String()], -1)
	}

	insertQ = fmt.Sprintf("insert into %s (%s)", curPartTbl, query)

	partTblQ = fmt.Sprintf("insert into %s values (%d, %d, $$%s$$, %d, $$%s$$, $$%s$$)", s.partTablename, now, s.output.Size, dbs.GetViewHostname(), dbs.GetViewPort(), dbs.GetViewDBname(), curPartTbl)

	return
}

func (s *JobSeries) sendAndWaitForJob(nextInputTime int64) (job remote.Job, token string) {
	job, execReply := schedConn.SendAndWaitForJob(s.output.Name, s.token, nextInputTime)
	s.jobStartTime = time.Now()
	return job, execReply.JobToken
}

func (s *JobSeries) sendJobDone(j remote.Job, token string, affectedRows int64) {
	_ = schedConn.SendJobDone(s.output.Name, token, time.Since(s.jobStartTime), affectedRows)
}
