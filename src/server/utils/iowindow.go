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
package utils

import (
	"log"
	"fmt"
	"strings"
	"strconv"
)

type IOWindow struct {
	Name          string
	PartTablename string
	Size          int
	Delay         int
	Primary       bool
}

func (w IOWindow) NextStartTs(currentTime int64) int64 {
	currentTime -= currentTime%int64(w.Size)
	return currentTime-int64(w.Size+w.Delay)
}

func (w IOWindow) NextEndTs(currentTime int64) int64 {
	currentTime -= currentTime%int64(w.Size)
	return currentTime-int64(w.Delay)-1
}

func (w IOWindow) String() string {
	return fmt.Sprintf("%s_%s_%d_%d_%t", w.Name, w.PartTablename, w.Size, w.Delay, w.Primary)
}

func GetWindowNames(ws []IOWindow) (names []string) {
	names = make([]string, len(ws))
	for i := 0; i < len(ws); i++ {
		names[i] = ws[i].Name
	}
	return names
}

func (wa IOWindow) Equals(wb IOWindow) bool {
	return wa.Name == wb.Name &&
		wa.PartTablename == wb.PartTablename &&
		wa.Size == wb.Size &&
		wa.Delay == wb.Delay &&
		wa.Primary == wb.Primary
}

func IOWindowFromString(wndDef string) (w IOWindow) {
	def := strings.ToLower(wndDef)
	wndStart := strings.Index(def, "(")
	wndEnd := strings.Index(def, ")")
	if wndStart < 0 || wndEnd < 0 {
		log.Fatalf("SYNTAX ERROR: window start or end is missing in definition: \"%s\".\nA window definition should e.g. look like this: \"tablename (window 120 delay 60 primary)\"\n", def)
	}
	wnd := strings.TrimSpace(def[wndStart+1 : wndEnd])
	name := strings.TrimSpace(def[:wndStart])

	fields := strings.Fields(wnd)
	if len(fields) > 5 || len(fields) < 2 {
		log.Fatalf("SYNTAX ERROR: Window definition: \"%s\"", wndDef)
	}
	wndSize := 0
	wndOffset := 0
	wndPrimary := false
	for i := 0; i < len(fields); i++ {
		var err error
		if strings.ToLower(fields[i]) == "window" {
			wndSize, err = strconv.Atoi(fields[i+1])
			i++
		} else if strings.ToLower(fields[i]) == "delay" {
			wndOffset, err = strconv.Atoi(fields[i+1])
			i++
		} else if strings.ToLower(fields[i]) == "primary" {
			wndPrimary = true
		} else {
			log.Fatalf("SYNTAX ERROR: Token \"%s\" not allowed in window definition: \"%s\".", fields[i], wndDef)
		}
		if err != nil {
			log.Fatalf("SYNTAX ERROR: \"%s\" not followed by a integer in window definition \"%s\".%v\n", fields[i], wndDef, err)
		}
	}

	w = IOWindow{name, "", wndSize, wndOffset, wndPrimary}
	return
}
