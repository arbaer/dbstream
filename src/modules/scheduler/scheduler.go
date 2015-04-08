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

import (
	"encoding/xml"
	"flag"
	"strings"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"sort"
	"time"
	"container/list"

	"./remote"
	"./stats"
	rep "../../server/lib/reporting"
	"../../server/lib/dbs"
	"../../server/utils"
)

/*
* TODO: Code cleanup needed
*/

type Config struct {
	XMLName      xml.Name	 `xml:"config"`
	Ip           string		 `xml:"ip,attr"`
	Port         string		 `xml:"port,attr"`
	ParallelJobs int		 `xml:"parallelJobs,attr"`
	Strategy	 string		 `xml:"strategy,attr"`
	DbsCfg		 dbs.DbsConn `xml:"dbs"`
}

type JobSeries struct {
	name     string
	description string
	regToken string
	inputs   []utils.IOWindow
	outputs  []utils.IOWindow

	inputTime    int64
	outputOffset int

	lastExecTime time.Duration 
}

type Scheduler struct {
	name   string
	cfg    Config
	series map[string]JobSeries
	client *rpc.Client

	waitingJobs map[string]remote.JobExec
	runningJobs map[string]remote.JobExec

	RegSeriesChan chan remote.SchedRegRequest
	RemSeriesChan chan string
	JobChan     chan remote.JobExec
	JobDoneChan chan remote.ExecStats

	queue		list.List
}

func readConfig() (cfg Config) {
	//decode the xml config file
	cfgFile, err := os.Open(*configFileName)
	if err != nil {
		log.Fatalf("ERROR while reading config: %v\n", err)
	}
	decode := xml.NewDecoder(cfgFile)
	err = decode.Decode(&cfg)
	if err != nil {
		log.Fatalf("ERROR while decoding config: %v\n", err)
	}
	cfgFile.Close()
	return
}

var configFileName = flag.String("config", "serverConf.xml", "The configuration file used in the server. If not present, the file 'serverConf.xml' will be used.")
var name = flag.String("name", "", "The name of the module.")
var serverLoc = flag.String("server", "", "The server location to report the status back.")

func main() {
	log.SetFlags(log.Flags() | 16)
	flag.Parse()
	cfg := readConfig()

	err := rep.Configure(*serverLoc)
	if err != nil {
		log.Fatalf("ERROR: while configuring reporting: %v", err)
	}

	err = dbs.Configure(cfg.DbsCfg)
	if err != nil {
		log.Fatalf("ERROR while configuring TicketDB: %v", err)
	}

	err = stats.Configure()
	if err != nil {
		log.Fatalf("ERROR: configuring stats collector: %v", err)
	}

	//initialize scheduler
	sched := new(Scheduler)
	sched.name = *name
	sched.cfg = cfg
	sched.RegSeriesChan = make(chan remote.SchedRegRequest)
	sched.RemSeriesChan = make(chan string)

	sched.JobChan = make(chan remote.JobExec)
	sched.JobDoneChan = make(chan remote.ExecStats)
	sched.series = make(map[string]JobSeries)

	sched.waitingJobs = make(map[string]remote.JobExec)
	sched.runningJobs = make(map[string]remote.JobExec)

	go sched.runRemote()
	go sched.sendStatus()

	sched.Run()
}

//Execute the scheduler
func (s *Scheduler) Run() {
	for {
		select {
		case req := <-s.RegSeriesChan:
			s.registerJobSeries(req)
		case rem := <-s.RemSeriesChan:
			s.removeSeries(rem)
		case execReq := <-s.JobChan:
			s.scheduleJob(execReq)
		case done := <-s.JobDoneChan:
			s.finalizeJob(done)
		}
	}
}

func (s *Scheduler) removeSeries(name string) {
	log.Printf("Removing Series %s...\n", name)
	s.removeJobForSeries(name)
	delete(s.series, name)
}

func (s *Scheduler) removeJobForSeries(seriesName string) {
	if _, found := s.runningJobs[seriesName]; found {
		delete(s.runningJobs, seriesName)
	}
	if _, found := s.waitingJobs[seriesName]; found {
		delete(s.waitingJobs, seriesName)
	}
}

func (s *Scheduler) sendStatus() {
	for {
		time.Sleep(1 * time.Second)
		_, err := rep.SendReport(s.name, s.Status())
		if err != nil {
			log.Fatal("ERROR: reporting status:", err)
		}
	}
}

func (s *Scheduler) scheduleJob(req remote.JobExec) {
	series, found := s.series[req.JobSeriesName]
	if !found {
		log.Printf("Job series %v not registered, rejecting.\n", req.JobSeriesName)
		req.ExecChan <- remote.ExecReply{JobToken: "", Execute: false}
		return
	}
	if series.regToken != req.RegToken {
		log.Printf("Job %v got an invalid token, rejecting.\n", req.Job.Name)
		delete(s.series, req.JobSeriesName)
		req.ExecChan <- remote.ExecReply{JobToken: "", Execute: false}
		return
	}
	if _, found := s.waitingJobs[req.JobSeriesName]; found {
		log.Printf("Job series %v has already a waiting job, rejecting.\n", req.JobSeriesName)
		req.ExecChan <- remote.ExecReply{JobToken: "", Execute: false}
		return
	}
	if _, found := s.runningJobs[req.JobSeriesName]; found {
		log.Printf("Job series %v has already a running job, rejecting.\n", req.JobSeriesName)
		req.ExecChan <- remote.ExecReply{JobToken: "", Execute: false}
		return
	}
	s.waitingJobs[req.JobSeriesName] = req
	s.executeJobs()
}

func (s *Scheduler) getLastInputTime(iname string) (t int64, found bool) {
	t = -1
	for _, ser := range s.series {
		for _, out := range ser.outputs {
			if out.Name == iname {
				t = ser.inputTime - int64(ser.outputOffset)
			}
		}
	}
	return t, t != -1
}


func (s *Scheduler) executeJobs() {
	switch s.cfg.Strategy {
	case "shared":
		s.executeJobsShared()
	case "freshness":
		s.executeJobsFreshness()
	case "fifo":
		s.executeJobsFifo()
	default:
		log.Printf("WARNING: strategy %s not implemented, using FIFO instead.", s.cfg.Strategy)
		s.executeJobsFifo()
	}
}

func (s *Scheduler) findJobCandidates() (candJobs []remote.JobExec) {
	candJobs = make([]remote.JobExec, 0)
	for name, job := range s.waitingJobs {
		var j *remote.JobExec
		j = nil
		ser := s.series[name]
		if len(ser.inputs) == 0 {
			j = &job
		} else {
			for _, input := range ser.inputs {
				t, found := s.getLastInputTime(input.Name)
				if !found {
					log.Printf("Input %s for jobSeries %s is missing.\n", input.Name, name)
					j = nil
					break
				} else {
					if t >= job.InputTime - int64(input.Delay) {
						j = &job
					} else {
						j = nil
						break
					}
				}
			}
		}
		if j != nil {
			log.Printf("Adding: %v", *j)
			candJobs = append(candJobs, job)
		}
	}
	//order jobs by the time they have been waiting
	curCandJobs := ""
	for _, c := range candJobs {
		curCandJobs += fmt.Sprintf(" %s:%d ,inps: %v ", c.Name, c.InputTime, s.series[c.JobSeriesName].inputs)
	}
	log.Printf("CandJobs: %v", curCandJobs)
	return candJobs
}

func (s *Scheduler) stalenessForSeries(seriesName string) int64 {
	maxInputTime := int64(0)
	for _, ser := range s.series {
		if ser.inputTime > maxInputTime {
			maxInputTime = ser.inputTime
		}
	}
	return maxInputTime - s.series[seriesName].inputTime
}

func (s *Scheduler) execTimeForSeries(seriesName string) float64 {
	return s.series[seriesName].lastExecTime.Seconds()
}

type JobsByFreshInc struct {
	s Scheduler
	Jobs
}

func (jobs JobsByFreshInc) Less(i, j int) bool {
	freshIncI := float64(jobs.s.stalenessForSeries(jobs.Jobs[i].JobSeriesName)) /
					jobs.s.execTimeForSeries(jobs.Jobs[i].JobSeriesName)
	freshIncJ := float64(jobs.s.stalenessForSeries(jobs.Jobs[j].JobSeriesName)) /
					jobs.s.execTimeForSeries(jobs.Jobs[j].JobSeriesName)
	//We want to sort by the highest freshness incrrease
	return freshIncI > freshIncJ
}


func (s *Scheduler) executeJobsFreshness() {
	log.Printf("Running Execute with strategy %s.", s.cfg.Strategy)
	if len(s.runningJobs) < s.cfg.ParallelJobs {
		//get series with executable jobs
		candJobs := s.findJobCandidates()
		sort.Sort(JobsByFreshInc{*s, candJobs})

		//Debug output
		debugFresh := "DebugFreshness: "
		for _, cj := range candJobs {

			staleN := s.stalenessForSeries(cj.JobSeriesName)
			execT := s.execTimeForSeries(cj.JobSeriesName)
			debugFresh = fmt.Sprintf("%s (%s s:%d, e:%f, res:%f) ", debugFresh, cj.Name,
				staleN, execT, float64(staleN) / execT)
		}
		log.Println(debugFresh)

		for i := 0; i < len(candJobs) && len(s.runningJobs) < s.cfg.ParallelJobs; i++ {
			//check for other jobs with the same input
			j := candJobs[i]
			//execute job
			token, err := utils.GenSecToken()
			if err != nil {
				log.Printf("Error while generating token: %v\n", err)
			}
			j.JobToken = token
			s.runningJobs[j.JobSeriesName] = j
			delete(s.waitingJobs, j.JobSeriesName)
			j.ExecChan <- remote.ExecReply{JobToken: token, Execute: true, Job: j.Job}
		}
	}
}
func (s *Scheduler) executeJobsShared() {
	log.Printf("Running Execute with strategy %s.", s.cfg.Strategy)
	if len(s.runningJobs) < s.cfg.ParallelJobs {
		//get series with executable jobs
		candJobs := s.findJobCandidates()

		for i := 0; i < len(candJobs) && len(s.runningJobs) < s.cfg.ParallelJobs; i++ {
			//check for other jobs with the same input
			postpone := false
			j := candJobs[i]
			ser := s.series[j.JobSeriesName]
			for _, serCur := range s.series {
				//only check other series
				if serCur.name != ser.name {
					for _, serIn := range serCur.inputs {
						for _, execIn := range ser.inputs {
							if serIn.Equals(execIn) && serCur.inputTime < ser.inputTime {
								postpone = true
								log.Printf("postponing: exec: %d: %v other: %d: %v", 
									ser.inputTime, execIn, serCur.inputTime, serIn)
							}
						}
					}
				}
			}
			if postpone {
				continue
			}

			token, err := utils.GenSecToken()
			if err != nil {
				log.Printf("Error while generating token: %v\n", err)
			}
			j.JobToken = token
			s.runningJobs[j.JobSeriesName] = j
			delete(s.waitingJobs, j.JobSeriesName)
			j.ExecChan <- remote.ExecReply{JobToken: token, Execute: true, Job: j.Job}
		}
	}
}

type Jobs []remote.JobExec
func (s Jobs) Len() int      { return len(s) }
func (s Jobs) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
type JobsByTime struct{ Jobs }
func (s JobsByTime) Less(i, j int) bool { return s.Jobs[i].RequestTime.Before(s.Jobs[j].RequestTime) }

func (s *Scheduler) executeJobsFifo() {
	log.Printf("Running Execute with strategy %s.", s.cfg.Strategy)
	if len(s.runningJobs) < s.cfg.ParallelJobs {
		//get series with executable jobs
		candJobs := s.findJobCandidates()
		sort.Sort(JobsByTime{candJobs})

		for i := 0; i < len(candJobs) && len(s.runningJobs) < s.cfg.ParallelJobs; i++ {
			//check for other jobs with the same input
			j := candJobs[i]
			//execute job
			token, err := utils.GenSecToken()
			if err != nil {
				log.Printf("Error while generating token: %v\n", err)
			}
			j.JobToken = token
			s.runningJobs[j.JobSeriesName] = j
			delete(s.waitingJobs, j.JobSeriesName)
			j.ExecChan <- remote.ExecReply{JobToken: token, Execute: true, Job: j.Job}
		}
	}
}

func (s *Scheduler) finalizeJob(stat remote.ExecStats) {
	sname := stat.JobSeriesName
	rj, found := s.runningJobs[sname]
	if !found {
		log.Printf("Job for jobSeries %s was not found in running jobs\n", sname)
	}
	if !(rj.JobToken == stat.JobToken) {
		log.Printf("Job for jobSeries %s has an invalid token %s != %s.\n", sname, rj.JobToken, stat.JobToken)
	}
	ser, found := s.series[sname]
	if found {
		ser.inputTime = rj.InputTime
		ser.lastExecTime = stat.ExecTime + rj.Overhead
		s.series[sname] = ser
	}
	log.Printf("Overhead: %v", rj.Overhead)

	if stat.Additional == nil {
		stat.Additional = make(map[string]string)
	}
	stat.Additional["description"] = ser.description
	stat.Additional["sched_strategy"] = s.cfg.Strategy
	stat.Additional["sched_parallelJobs"] = fmt.Sprintf("%d", s.cfg.ParallelJobs)

	stats.Collect(time.Now(),
		sname,
		rj.Name,
		strings.Join(utils.GetWindowNames(ser.inputs), ", "),
		strings.Join(utils.GetWindowNames(ser.outputs), ", "),
		ser.inputTime,
		stat.ExecTime + rj.Overhead,
		stat.AffectedRows,
		stat.Additional)
	delete(s.runningJobs, sname)

	s.executeJobs()
}

func (s *Scheduler) registerJobSeries(req remote.SchedRegRequest) {
	series, found := s.series[req.JobSeriesName]
	if found {
		log.Printf("Job series %v was already registered, rejecting.\n", series.name)
		req.RegReplyChan <- remote.RegReply{JobSeriesName: req.JobSeriesName, Success: false, RegToken: ""}
		return
	}
	token, err := utils.GenSecToken()
	if err != nil {
		log.Printf("Error while generating token: %v\n", err)
	}
	newSeries := JobSeries{req.JobSeriesName,
		req.Description,
		token,
		req.Inputs,
		req.Outputs,
		req.InputTime,
		req.OutputOffset,
		time.Duration(1)*time.Second}
	s.series[req.JobSeriesName] = newSeries
	req.RegReplyChan <- remote.RegReply{JobSeriesName: req.JobSeriesName, Success: true, RegToken: newSeries.regToken}
	s.executeJobs()
}

func (s *Scheduler) Status() (status map[string]string) {
	status = make(map[string]string)
	status["General"] = fmt.Sprintf("%v series currently registered. Jobs: %d waiting %d running %d idle", len(s.series), len(s.waitingJobs), len(s.runningJobs), len(s.series)-len(s.waitingJobs)-len(s.runningJobs))
	sortSers := make([]string, len(s.series))
	i := 0
	maxLen := 0
	for name, _ := range s.series {
		if len(name) > maxLen {
			maxLen = len(name)
		}
		sortSers[i] = name
		i++
	}
	sort.Strings(sortSers)
	for _, name := range sortSers {
		ident := ""
		for i := 0; i < maxLen-len(name); i++ {
			ident += " "
		}
		state := "idle"
		if _, running := s.runningJobs[name]; running {
			state = "running"
		}
		if _, waiting := s.waitingJobs[name]; waiting {
			state = "waiting"
		}
		status[name] = fmt.Sprintf("%sInputTime %s Offset %d execTime %s  State %s", ident, time.Unix(s.series[name].inputTime, 0), s.series[name].outputOffset, s.series[name].lastExecTime, state)
	}
	return status
}

func (s Scheduler) runRemote() {

	l, e := net.Listen("tcp", s.cfg.Ip+":"+s.cfg.Port)
	if e != nil {
		log.Fatal("ERROR: listen error:", e)
	}
	for {
		log.Println("Serving....")
		c, err := l.Accept()
		if err != nil {
			log.Fatalf("ERROR: Accept Error: %v", err)
		}
		go s.runConn(c)
	}
}

func (s Scheduler) runConn(c net.Conn) {
	schedRem := new(remote.SchedRemote)

	schedRem.RegSeriesChan = make(chan remote.SchedRegRequest)
	schedRem.JobChan = s.JobChan
	schedRem.JobDoneChan = s.JobDoneChan

	server := rpc.NewServer()
	server.Register(schedRem)
	exitChan := waitForConnExit(c, server)

	series := make(map[string]bool)

	for {
		select {
		case req := <-schedRem.RegSeriesChan:
			replyChan := req.RegReplyChan
			req.RegReplyChan = make(chan remote.RegReply)
			s.RegSeriesChan <- req
			reply := <-req.RegReplyChan
			if reply.Success {
				series[reply.JobSeriesName] = true
			}
			replyChan <- reply
		case <-exitChan:
			for name, _ := range series {
				s.RemSeriesChan <- name
				delete(series, name)
			}
		}
	}
}

func waitForConnExit(c net.Conn, server *rpc.Server) (ret chan bool) {
	ret = make(chan bool)
	go func() {
		tcpConn := c.(*net.TCPConn)
		tcpConn.SetKeepAlive(true)
		server.ServeConn(c)
		ret <- true
	}()
	return ret
}
