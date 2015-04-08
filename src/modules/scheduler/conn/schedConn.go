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
package conn

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/rpc"
	"time"

	"../remote"
	"../../../server/utils"
)

var scheduler *rpc.Client

type SchedulerCfg struct {
	XMLName xml.Name `xml:"scheduler"`
	Ip      string   `xml:"ip,attr"`
	Port    string   `xml:"port,attr"`
}

func Configure(cfg SchedulerCfg) (err error) {
	scheduler, err = rpc.Dial("tcp", cfg.Ip+":"+cfg.Port)
	if err != nil {
		log.Panicf("ERROR while connecting to scheduler: %s", err)
	}
	go WaitForConnClose(scheduler)
	return
}

func WaitForConnClose(client *rpc.Client) {
	var in, out bool
	err := client.Call("SchedRemote.WaitForConnClose", &in, &out)
	if err != nil {
		log.Panic("ERROR: Connection closed, exiting...")
	}
}

func RegisterJobSeries(name, description string, inputs []utils.IOWindow, outputs []utils.IOWindow, inputTime int64, outputOffset int) (reply remote.RegReply) {

	req := &remote.RegRequest{JobSeriesName: name, Description: description, Inputs: inputs, Outputs: outputs, InputTime: inputTime, OutputOffset: outputOffset}
	err := scheduler.Call("SchedRemote.RegisterJobSeries", req, &reply)
	if err != nil {
		log.Panic("ERROR: SchedConn error:", err)
	}
	log.Printf("JobSeries %v registered successfully: %+v\n", name, reply)
	return
}

func SendAndWaitForJob(name string, seriesToken string, nextInputTime int64) (job remote.Job, execReply remote.ExecReply) {
	jname := fmt.Sprintf("%s_%d", name, nextInputTime)
	job = remote.Job{Name: jname, Description: "", InputTime: nextInputTime}
	execReq := remote.ExecRequest{JobSeriesName: name, RegToken: seriesToken, Job: job}

	err := scheduler.Call("SchedRemote.WaitForJobExec", execReq, &execReply)
	if err != nil {
		log.Panicf("ERROR: Send Job Failed: %v\n", err)
	}

	//log.Printf("Job %s ready for execute, reply: %+v\n", jname, execReply)
	return execReply.Job, execReply
}

func SendJobDone(name string, token string, duration time.Duration, affectedRows int64) (ok bool) {
	stats := remote.ExecStats{JobSeriesName: name, JobToken: token, ExecTime: duration, AffectedRows: affectedRows}
	err := scheduler.Call("SchedRemote.JobDone", stats, &ok)
	if err != nil {
		log.Printf("sendJobDone failed.\n%v\n", err)
	}
	return
}
