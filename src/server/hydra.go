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
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	rep "./lib/reporting"
	"./utils"
)

var configFileName = flag.String("config", "serverConf.xml", "The configuration file used in the server. If not present, the file 'serverConf.xml' will be used.")
var debug = flag.Bool("debug", false, "Endables debug functionallity.")
var logFile *os.File = nil
var logWriter io.Writer

type Log struct {
	Type string `xml:"type,attr"`
	File string `xml:"file,attr"`
}

//This is the configuration of the server
type Config struct {
	XMLName        xml.Name       `xml:"config"`
	Ip             string         `xml:"ip,attr"`
	Port           string         `xml:"port,attr"`
	RestartPenalty int            `xml:"restartPenalty,attr"`
	Log            Log            `xml:"log"`
	Modules        []utils.Module `xml:"modules>module"`
}

type Server struct {
	cfg              Config
	mainExitChan     chan bool
	logChan          chan string
	exitChan         chan utils.Module
	checkModChan     chan bool
	sigChan          chan os.Signal
	reportChan       chan rep.ModReport
	requestStatsChan chan bool
	statsChan        chan map[string]utils.Module

	runningModules map[string]*utils.Module

	lastStatus time.Time
}

type ServerError string

func (e ServerError) Error() string {
	return string(e)
}

func readConfig() (cfg Config) {
	//decode the xml config file
	cfgFile, err := os.Open(*configFileName)
	if err != nil {
		log.Fatalf("ERROR: Config file Error: %v\n", err)
	}
	decode := xml.NewDecoder(cfgFile)
	err = decode.Decode(&cfg)
	if err != nil {
		log.Printf("Configuration Error: %v\n", err)
	}
	cfgFile.Close()
	return
}

func main() {
	runtime.GOMAXPROCS(4)
	flag.Parse()
	if flag.Lookup("config").Value.String() == flag.Lookup("config").DefValue {
		log.Printf("Using default config file %v\n", flag.Lookup("config").DefValue)
	}

	var s Server

	cfg := readConfig()
	s.cfg = cfg
	logConf(cfg.Log)

	//create syncronising channels
	s.mainExitChan = make(chan bool)
	s.logChan = make(chan string, 10)
	s.exitChan = make(chan utils.Module)
	s.checkModChan = make(chan bool)
	//register signals
	s.sigChan = make(chan os.Signal)
	s.reportChan = make(chan rep.ModReport, 10)
	s.requestStatsChan = make(chan bool)
	s.statsChan = make(chan map[string]utils.Module)
	signal.Notify(s.sigChan)
	go s.handleSig()

	s.runningModules = make(map[string]*utils.Module)

	//register remote
	remote := new(utils.Remote)
	remote.RequestStatsChan = s.requestStatsChan
	remote.StatsChan = s.statsChan
	remote.CheckModChan = s.checkModChan
	remote.ReportChan = s.reportChan
	go runRemote(remote, cfg.Ip, cfg.Port)

	//start modules
	for _, module := range cfg.Modules {
		err := s.startModule(module)
		if err != nil {
			log.Fatalf("ERROR: can not configure module %v\n", err)
		}
	}

	s.mainLoop()
}

func runRemote(r *utils.Remote, ip string, port string) {
	if len(ip) == 0 {
		ip = "localhost"
	}
	if len(port) == 0 {
		port = "5322"
	}
	rpc.Register(r)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", ip+":"+port)
	if err != nil {
		log.Fatal("ERROR: Can not register remote interface", err)
	}
	for {
		http.Serve(l, nil)
	}
}

func (s *Server) startModule(cfgMod utils.Module) (err error) {
	if len(cfgMod.Name) == 0 {
		return ServerError(fmt.Sprintf("Module does not have a Name: %s", cfgMod.Config))
	}
	if strings.IndexAny(cfgMod.Name, " \n\t\r") > 0 {
		return ServerError(fmt.Sprintf("Module name: \"%s\" incorrect. Module names are not allowed to contain space, newline or tab.", cfgMod.Name))
	}
	if len(cfgMod.Executable) == 0 {
		return ServerError(fmt.Sprintf("Module does not have a Executable: %s", cfgMod.Config))
	}
	if _, exists := s.runningModules[cfgMod.Name]; exists {
		return ServerError(fmt.Sprintf("Module with name \"%v\" is configured more than once, module names have to be unique.\n", cfgMod.Name))
	}

	mod := new(utils.Module)
	*mod = cfgMod
	mod.Stats = new(utils.ModStat)
	if mod.RestartPenalty == 0 {
		mod.RestartPenalty = s.cfg.RestartPenalty
	}
	s.runningModules[mod.Name] = mod
	log.Printf("Starting module: %v\n", mod.Name)
	serverLoc := s.cfg.Ip + ":" + s.cfg.Port
	err = mod.Start(serverLoc, s.logChan, s.exitChan)
	return err
}

func (s *Server) mainLoop() {
	//create a logger for the modules
	modLog := log.New(logWriter, "", 0)
	s.lastStatus = time.Now()
	for {
		select {
		case cmod := <-s.exitChan: //wait for exiting modules
			//check if the main program will exit as well to avoid unnessesary module restarts
			modCrash := make(map[string]string)
			modCrash["ERROR"] = "This Module crashed, please see the server log for details."
			s.runningModules[cmod.Name].Stats = &utils.ModStat{ReportTime: time.Now(), Values: modCrash}
			select {
			case _ = <-s.mainExitChan:
				s.mainExitChan <- true
			case <-time.After(time.Second):
				go s.restartModule(cmod)
			}
		case line := <-s.logChan: //wait for log messages
			modLog.Printf("%s", line)
		case _ = <-s.checkModChan:
			s.startNewModules()
		case report := <-s.reportChan:
			s.UpdateModStatus(report)
		case <-s.requestStatsChan:
			s.SendStats()
		case <-time.After(1 * time.Second):
		}
		//print status message
		if *debug {
			if time.Since(s.lastStatus) >= 1*time.Second {
				for _, module := range s.runningModules {
					log.Println(module.Status())
				}
				s.lastStatus = time.Now()
			}
		}
	}
}

func (s *Server) SendStats() {
	mods := make(map[string]utils.Module)
	for name, mod := range s.runningModules {
		mods[name] = *mod
	}
	s.statsChan <- mods
}

func (s *Server) UpdateModStatus(report rep.ModReport) {
	m := s.runningModules[report.ModName]
	stat := utils.ModStat{ReportTime: time.Now(), Values: report.Values}
	*m.Stats = stat
}

func (s *Server) restartModule(crashedMod utils.Module) {
	mod := s.runningModules[crashedMod.Name]

	//reread config to check for changes for this module
	s.cfg = readConfig()
	var cfgMod utils.Module
	found := false
	for _, m := range s.cfg.Modules {
		if m.Name == mod.Name {
			cfgMod = m
			found = true
		}
	}
	if !found {
		//config does not contain this module anymore
		log.Printf("Module %v is not configured anymore and will not be restarted.\n", mod.Name)
		delete(s.runningModules, mod.Name)
	} else {
		if mod.Running {
			//module is still configured and will be restarted with new config
			log.Printf("Module %v crashed", mod.Name)
			mod.Executable = cfgMod.Executable
			mod.Config = cfgMod.Config
			mod.Arguments = cfgMod.Arguments
			s.runningModules[mod.Name] = mod
			serverLoc := string(s.cfg.Ip) + ":" + string(s.cfg.Port)
			mod.Restart(serverLoc, s.logChan, s.exitChan)
			statRestart := make(map[string]string)
			statRestart["RESTART"] = "Module is restarting."
			mod.Stats = &utils.ModStat{ReportTime: time.Now(), Values: statRestart}
		} else {
			log.Printf("Module %v will not be restarted.\n", mod.Name)
		}
	}
}

func (s *Server) startNewModules() {
	s.cfg = readConfig()

	for _, m := range s.cfg.Modules {
		if _, exists := s.runningModules[m.Name]; !exists {
			err := s.startModule(m)
			if err != nil {
				log.Printf("Error while starting module \"%v\": %v\n", m.Name, err)
			}
		}
	}
}

func (s *Server) handleSig() {
	for sig := range s.sigChan {
		if sig == syscall.SIGKILL || sig == syscall.SIGINT || sig == syscall.SIGTERM {
			s.mainExitChan <- true
			s.exit()
		} else if sig == syscall.SIGUSR1 {
			log.Println("Checking for new modules.")
			s.checkModChan <- true
		} else {
			log.Printf("Handling signal: %v\n", sig)
		}
	}
}

func (s *Server) exit() {
	for _, mod := range s.runningModules {
		log.Printf("Shutting down module %v.\n", mod.Name)
		mod.Running = false
		log.Printf("Module %v exited\n", mod.Name)
	}
	//empty the log
	for {
		select {
		case line := <-s.logChan:
			log.Println(line)
		case <-time.After(time.Second):
		}
		break
	}
	log.Println("Main exit!")
	if logFile != nil {
		logFile.Close()
	}
	os.Exit(0)
}

func logConf(cfg Log) {
	logWriter = os.Stderr
	if cfg.Type != "" {
		writers := make([]io.Writer, 0)
		if strings.Index(cfg.Type, "file") >= 0 {
			logFilename := cfg.File
			logFile, err := os.Create(logFilename)
			if err != nil {
				log.Fatalf("ERROR: Could not open logfile %v\n%v", logFilename, err)
			}
			writers = append(writers, logFile)
		}
		if strings.Index(cfg.Type, "stderr") >= 0 {
			writers = append(writers, os.Stderr)
		}
		if strings.Index(cfg.Type, "stdout") >= 0 {
			writers = append(writers, os.Stdout)
		}
		if len(writers) > 0 {
			logWriter = io.MultiWriter(writers...)
		}
	} else {
		log.Printf("no logging config found using stderr.\n")
	}
	log.SetOutput(logWriter)
	log.SetFlags(19)
}
