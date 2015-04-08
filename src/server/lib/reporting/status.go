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
package reporting

import (
	"net/rpc"
)

var server *rpc.Client

type ModReport struct {
	ModName string
	Values  map[string]string
}

func Configure(serverLoc string) (err error) {
	server, err = rpc.DialHTTP("tcp", serverLoc)
	return err
}

func SendReport(name string, status map[string]string) (reply bool, err error) {
	req := &ModReport{name, status}
	err = server.Call("Remote.ReportStatus", req, &reply)
	return reply, err
}
