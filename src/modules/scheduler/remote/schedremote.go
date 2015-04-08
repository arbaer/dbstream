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
package remote

import (
	"time"

	"../../../server/utils"
)

type RegRequest struct {
	JobSeriesName string
	Description	  string
	Inputs        []utils.IOWindow
	Outputs       []utils.IOWindow
	InputTime     int64
	OutputOffset  int
}

type RegReply struct {
	JobSeriesName string
	Success       bool
	RegToken      string
}

type JobExec struct {
	ExecRequest
	Overhead time.Duration
	RequestTime time.Time
	JobToken    string
	ExecChan    chan ExecReply
}

type ExecRequest struct {
	JobSeriesName string
	RegToken      string
	Job
}

type Job struct {
	Name        string
	Description string
	InputTime   int64
}

type ExecReply struct {
	JobToken string
	Execute  bool
	Job
}

type ExecStats struct {
	JobSeriesName string
	JobToken      string
	ExecTime      time.Duration
	AffectedRows  int64
	Additional	  map[string]string
}

/*
	Internal data Structures for the Scheduler remote interface
*/
type SchedRemote struct {
	RegSeriesChan chan SchedRegRequest
	RegAckChan    chan RegReply

	JobChan     chan JobExec
	JobDoneChan chan ExecStats
}

type SchedRegRequest struct {
	RegRequest
	RegReplyChan chan RegReply
}

type SchedRemoteError string

func (e SchedRemoteError) Error() string {
	return string(e)
}

//remote function implementation
func (s *SchedRemote) RegisterJobSeries(req RegRequest, reply *RegReply) (err error) {
	replyChan := make(chan RegReply)
	schedReg := SchedRegRequest{req, replyChan}
	s.RegSeriesChan <- schedReg
	*reply = <-replyChan
	if !reply.Success {
		return SchedRemoteError("Job Series was already registered, rejecting.")
	}
	return nil
}

func (s *SchedRemote) WaitForJobExec(req ExecRequest, reply *ExecReply) (err error) {
	execJob := JobExec{req, 0, time.Now(), "", make(chan ExecReply)}
	s.JobChan <- execJob
	*reply = <-execJob.ExecChan
	if !reply.Execute {
		return SchedRemoteError("Invalid Token")
	}
	return nil
}

func (s *SchedRemote) JobDone(stats *ExecStats, reply *bool) (err error) {
	s.JobDoneChan <- *stats
	*reply = true
	return nil
}

func (s *SchedRemote) WaitForConnClose(in *bool, out *bool) error {
	c := make(chan bool)
	<-c
	return nil
}
