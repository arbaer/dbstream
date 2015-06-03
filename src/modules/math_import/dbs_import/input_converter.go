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
	"fmt"
	"log"
	"strconv"
	"strings"
)

/*
* InputConverter
*
* The InputConverter interface can be used to implement new input converters for different formats.
*/
type InputConverter interface {
	//convert one line of input data into an array of strings
	Line2Array(line string) []string
	//check if the line has all fields needed for the given format
	CheckLine(sline []string) (isValid bool)
	//extract time from one input line
	GetSerialTime(sline []string, lastSerialTime int64) (serialTime int64)	
	//convert line into Postgres representation
	ConvertLine(sline []string, delimiter string)  string
}

//method generating new InputConverter according to the input format given by the probe
func GetInputConverter(inputFormat string) InputConverter {
	if inputFormat == "log_tcp_complete_v15" {
		return TstatLogTCPV15{28,29," "}
	} else if inputFormat == "log_video_complete" {
		return TstatLogVideo{
			first_abs:67, 
			last:60, 
			inputDelimiter: " ",
		}
	}
	panic(fmt.Sprintf("Input type '%s' not implemented", inputFormat))
}

// TstatLogTCPV15 implementation
type TstatLogTCPV15 struct {
	first int
	last int
	inputDelimiter string
}

func (t TstatLogTCPV15) Line2Array(line string) []string {
	return strings.Split(line, " ")
}

func (t TstatLogTCPV15) CheckLine(sline []string) (isValid bool) {
	return len(sline) == 130
}

func (t TstatLogTCPV15) GetSerialTime(sline []string, lastSerialTime int64) (serialTime int64) {

	last, err := strconv.ParseFloat(sline[t.last], 64)
	if err != nil {
		log.Panic("ERROR: %s", err)
	}

	newTS := int64(last/1000)
	if newTS > lastSerialTime {
		return newTS
	} else {
		return lastSerialTime
	}
}

func (t TstatLogTCPV15) ConvertLine(sline []string, delimiter string)  string {
	return strings.Join(sline, delimiter)
}
// /TstatLogTCPV15 implementation



// TstatLogVideo implementation
type TstatLogVideo struct {
	first_abs int
	last int
	inputDelimiter string
}

func (t TstatLogVideo) Line2Array(line string) []string {
	return strings.Split(line, t.inputDelimiter)
}

func (t TstatLogVideo) CheckLine(sline []string) (isValid bool) {
	if len(sline) != 125 {
		log.Panic("LINE LEN: ", len(sline))
	}	
	return true
}

func (t TstatLogVideo) GetSerialTime(sline []string, lastSerialTime int64) (serialTime int64) {

	first_abs, err := strconv.ParseFloat(sline[t.first_abs], 64)
	if err != nil {
		log.Panic("ERROR: %s", err)
	}

	last, err := strconv.ParseFloat(sline[t.last], 64)
	if err != nil {
		log.Panic("ERROR: %s", err)
	}

	newTS := int64( (first_abs + last)/1000 )
	if newTS > lastSerialTime {
		return newTS
	} else {
		return lastSerialTime
	}
}

func (t TstatLogVideo) ConvertLine(sline []string, delimiter string)  string {
	return strings.Join(sline, delimiter)
}
// /TstatLogVideo implementation
