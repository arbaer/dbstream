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
	"regexp"
	"sort"

	rep "../../server/lib/reporting"
)

type Remote struct {
	RequestStatsChan chan bool
	StatsChan        chan map[string]Module
	CheckModChan     chan bool
	ReportChan       chan rep.ModReport
}

func (r *Remote) ListModules(filter *string, reply *[]string) (err error) {
	r.RequestStatsChan <- true
	mods := <-r.StatsChan
	mNames := make([]string, len(mods))
	i := 0
	for name, _ := range mods {
		mNames[i] = name
		i++
	}
	*reply = mNames
	return nil
}

type StatusFilter struct {
	ModFilter   string
	ValueFilter string
}

func (r *Remote) ListModulesStatus(filter *StatusFilter, reply *[]string) (err error) {
	r.RequestStatsChan <- true
	mods := <-r.StatsChan
	mStatus := make([]string, len(mods))	
	modNames := make([]string, len(mods))

	if filter == nil {
		filter = &StatusFilter{".*", ".*"}
	}
	if filter.ModFilter == "" {
		filter.ModFilter = ".*"
	}
	if filter.ValueFilter == "" {
		filter.ValueFilter = ".*"
	}

	i := 0
	for name, _ := range mods {
		found, err := regexp.MatchString(filter.ModFilter, name)
		if err != nil {
			log.Println(err)
			return err
		}
		if found {
			modNames[i] = name
			i++
		}
	}
	modNames = modNames[:i]
	mStatus = mStatus[:i]
	sort.Strings(modNames)
	for j, name := range modNames {
		mStatus[j] = name + ": " + mods[name].StatusFilt(filter.ValueFilter)
	}
	*reply = mStatus
	return nil
}

func (r *Remote) ReReadConfig(nop bool, nope *bool) (err error) {
	log.Println("Checking for new modules.")
	r.CheckModChan <- true
	return nil
}

func (r *Remote) ReportStatus(status *rep.ModReport, reply *bool) (err error) {
	r.ReportChan <- *status
	*reply = true
	return nil
}
