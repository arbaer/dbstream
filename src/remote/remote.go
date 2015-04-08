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
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"strconv"
	"syscall"
	"strings"

	"../server/utils"
)

var ipaddr = flag.String("ip", "localhost", "The IP address or hostname of the TicketDB 2.0 application server to query. If not present 'localhost' will be used.")
var port = flag.String("port", "5322", "The port of the of the TicketDB 2.0 application server. If not present the default port '5322' will be used.")
var modFilt = flag.String("modFilt", ".*", "Filter the modules, by applying the given regular expression on the module name.")
var valFilt = flag.String("valFilt", ".*", "Filter the values reported by modules, by applying the given regular expression on the value name.")
var checkConf = flag.Bool("checkconfig", false, "Issues the application server to reread the configuration, searching for new modules to start.")
var restart = flag.String("restart", "", "A module can be restarted by it's name.")
var ls = flag.Bool("ls", false, "List module names.")

func main() {
	flag.Parse()

	client, err := rpc.DialHTTP("tcp", fmt.Sprintf("%s:%s", *ipaddr, *port))
	if err != nil {
		log.Fatal("ERROR: Server Connection failed:", err)
	}

	var reply []string
	if *checkConf {
		var nop bool
		err = client.Call("Remote.ReReadConfig", nop, &nop)
		if err != nil {
			log.Fatal("ERROR: Call Failed:", err)
		}
		fmt.Println("Config re-read sent.")
	} else if *ls {
		err = client.Call("Remote.ListModules", "*", &reply)
		if err != nil {
			log.Fatal("ERROR: Call Failed:", err)
		}

		fmt.Println("Running Modules:")
		for _, m := range reply {
			fmt.Printf("\t%v\n", m)
		}
		fmt.Println()
	} else if *restart != "" {
		*valFilt = ""
		filter := utils.StatusFilter{ModFilter: *restart, ValueFilter: *valFilt}
		err = client.Call("Remote.ListModulesStatus", &filter, &reply)
		if err != nil {
			log.Fatal("ERROR: Call Failed:", err)
		}
		var pid int
		for _, m := range reply {
			if strings.Index(m, " PID: ") > 0 {
				parts := strings.Split(m, " ")
				if parts[0] != *restart + ":" {
					log.Fatal("ERROR: Module not found.")
				}
				pid, err = strconv.Atoi(parts[2])
				if err != nil {
					log.Fatalf("ERROR: can not parse PID, %v", err)
				}
				break
			}
		}
		if pid != 0 {
			fmt.Printf("Restarting module %s\n", *restart)
			syscall.Kill(pid, 15)
		} else {
			log.Fatal("ERROR: Module not found.")
		}
	} else {
		filter := utils.StatusFilter{ModFilter: *modFilt, ValueFilter: *valFilt}
		err = client.Call("Remote.ListModulesStatus", &filter, &reply)
		if err != nil {
			log.Fatal("ERROR: Call Failed:", err)
		}

		fmt.Println("Running Module Status:\n ")
		for _, m := range reply {
			fmt.Printf("%v\n\n", m)
		}
		fmt.Println()
	}
}
