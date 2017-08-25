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
package dbs

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	//"time"
	"math"
	
	//"context"
	"github.com/jackc/pgx"
)

type DbsConn struct {
	lock     sync.Mutex
	DbName   string `xml:"dbname,attr"`
	User     string `xml:"user,attr"`
	Port     int    `xml:"port,attr"`
	Host     string `xml:"host,attr"`
	Password string `xml:"password,attr"`
	//pool     *pgsql.Pool
	pool     *pgx.ConnPool
	prepareConn *pgx.Conn
}

//static variable for internal use only
var dbs DbsConn

func Configure(cfg DbsConn) (err error) {
	dbs = cfg

	dbs.lock.Lock()
	defer dbs.lock.Unlock()

	//conStr := fmt.Sprintf("dbname=%v host=%v port=%v user=%v password=%v",
	//	dbs.DbName, dbs.Host, dbs.Port, dbs.User, dbs.Password)
	log.Printf("dbname=%v host=%v port=%v user=%v\n", dbs.DbName, dbs.Host, dbs.Port, dbs.User)

	dbconfig := &pgx.ConnConfig{
					Host: dbs.Host, 
					User: dbs.User, 
					Password: dbs.Password, 
					Database: dbs.DbName}

	config := pgx.ConnPoolConfig{
		ConnConfig: *dbconfig, 
		MaxConnections: 10}
	pool, err := pgx.NewConnPool(config)

	//pool, err := pgsql.NewPool(conStr, 1, 10, time.Second*10)
	if err != nil {
		return err
	}

	dbs.pool = pool
	log.Printf("Connection to DbsMaster created.")
	return err
}

func Close() {
//	err := dbs.pool.Close()
//	if err != nil {
//		log.Fatal(err)
//	}
	dbs.pool.Close()
}

func GetViewHostname() string {
	return dbs.Host
}

func GetViewPort() int {
	return dbs.Port
}

func GetViewDBname() string {
	return dbs.DbName
}

func Execf(query string, args ...interface{}) (rowsAffected int64) {
	query = fmt.Sprintf(query, args...)
	return Exec(query)
}

func Exec(query string) (rowsAffected int64) {
	conn, err := dbs.pool.Acquire()
	if err != nil {
		log.Fatalf("Dbs Exec failed.\n%v", err)
	}
	defer dbs.pool.Release(conn)

	var ct pgx.CommandTag
	ct, err = conn.Exec(query)
	if err != nil {
		log.Fatalf("Query: '%v' failed to execute.", query)
	}
	rowsAffected = ct.RowsAffected()
	return
}

func ExecErr(query string) (rowsAffected int64, err error) {
	conn, err := dbs.pool.Acquire()
	if err != nil {
		log.Fatalf("Dbs Exec failed.\n%v", err)
	}
	defer dbs.pool.Release(conn)

	var ct pgx.CommandTag
	ct, err = conn.Exec(query)
	rowsAffected = ct.RowsAffected()


	return rowsAffected, err
}

func Prepare(name, query string) (conn *pgx.Conn, err error) {
	if dbs.prepareConn == nil {
		var err error
		dbs.prepareConn, err = dbs.pool.Acquire()
		if err != nil {
			log.Fatalf("Dbs Prepare failed.\n%v", err)
		}
	}

	_, err = dbs.prepareConn.Prepare(name, query)
	return dbs.prepareConn, err
}

type DbsTable struct {
	Tablename     string
	PartTablename string
	TblType       string
}

func GetTables() (tbls []DbsTable) {
	tbls = fetchTable("select * from dbs_tables;", reflect.TypeOf(tbls)).([]DbsTable)
	return
}

func GetTable(name string) (tbls []DbsTable) {
	query := fmt.Sprintf("select * from dbs_tables where tablename='%s'", name)
	tbls = fetchTable(query, reflect.TypeOf(tbls)).([]DbsTable)
	return
}

func GetMaxTimestampForTable(tablename string) (maxts int64) {
	query := fmt.Sprintf("select max(serial_time) from %v", escape(tablename))
	var res []int64
	res = fetchTable(query, reflect.TypeOf(res)).([]int64)
	return res[0]
}

func CreateViewParent(tblName, schema string) {
	rowsCnt := Execf("select tablename from pg_tables where tablename='%v'", tblName)
	if rowsCnt == 0 {
		rowsCnt = Execf("create table %v (%v);", tblName, schema)
	}
}

type DbsParttable struct {
	SerialTime int
	PartLength int
	Hostname   string
	Port       int
	Dbname     string
	Tablename  string
}

//checks if the partition table does already exist and creates it if not.
//In addition, the serial_time of the last successful import for this table or 0 is returned
func GetOrCreateJobState(tblName string) (partTblname string, curPartTblname string, lastImportTime int64) {
	//check if table exists
	tblName = escape(tblName)

	getPartnamesQ := fmt.Sprintf("select * from dbs_tables where tablename='%s'", escape(tblName))
	var part_names []DbsTable
	part_names = fetchTable(getPartnamesQ, reflect.TypeOf(part_names)).([]DbsTable)
	if len(part_names) == 0 {
		partTblname = fmt.Sprintf("dbs.view_info_%v", tblName)
		_ = Execf("insert into dbs_tables values ('%v', '%v', '%v')",
			escape(tblName), escape(partTblname), "view")
	} else {
		partTblname = part_names[0].PartTablename
	}
	lastImportTime = 0
	partTblnameNoSchema := strings.Split(partTblname, ".")[1];
	existsQ := fmt.Sprintf("select tablename from pg_tables where tablename='%v'", partTblnameNoSchema)
	rows := Exec(existsQ)
	var tbl []DbsParttable
	if rows > 0 {
		query := fmt.Sprintf("select * from %v order by serial_time desc limit 1", partTblname)
		tbl = fetchTable(query, reflect.TypeOf(tbl)).([]DbsParttable)
		if len(tbl) > 0 {
			lastImportTime = int64(tbl[0].SerialTime)
			curPartTblname = tbl[0].Tablename
		}
	} else {
		createQ := fmt.Sprintf(`create table %v (
		serial_time integer,
		part_length integer,
		hostname varchar(256),
		port integer,
		dbname varchar(256),
		tablename varchar(256)
		)`, partTblname)
		_ = Exec(createQ)
		_ = Execf("create index %v_serial_time_idx on %v (serial_time)", partTblnameNoSchema, partTblname)
		log.Printf("Part table %v created", partTblname)
	}

	return
}

func DeleteFromParttable(tblName string, startts , endts int64) {
	if endts == -1 {
		endts = math.MaxInt32
	}

	//get part tables
	partTQ := fmt.Sprintf("select distinct tablename from %v where serial_time >= %d and serial_time < %d;", tblName, startts, endts)
	var tbls []string
	tbls = fetchTable(partTQ, reflect.TypeOf(tbls)).([]string)
	query := "BEGIN;"
	for _, tbl := range tbls {
		query += fmt.Sprintf("drop table if exists %s cascade;", tbl)
	}
	query += fmt.Sprintf("delete from %v where serial_time >= %d and serial_time < %d;", tblName, startts, endts)

	query += " COMMIT;"
	log.Println("initDelQ: " + query)
	_ = Exec(query)
}

func GetNewParts(partTblname string, lastTs int) (parts []DbsParttable) {
	query := fmt.Sprintf("select * from %v where serial_time >= %v order by serial_time asc", escape(partTblname), lastTs)
	return fetchTable(query, reflect.TypeOf(parts)).([]DbsParttable)
}

type DbsPartSize struct {
	MinSerialTime int
	MaxSerialTime int
	Tablename  string
	Size int64
}

func GetPartSizeTimeSort(partTblname string) (parts []DbsPartSize, err error) {
	query := fmt.Sprintf("select *, pg_total_relation_size(tablename) from (select min(serial_time) as min_st, max(serial_time) as max_st, tablename from %s group by tablename order by 1) foo;", escape(partTblname))
	ret, err := safeFetchTable(query, reflect.TypeOf(parts))
	return ret.([]DbsPartSize), err
}

func GetSliceQuery(wndName, partTablename string, startts, endts int64) (query string) {
	q_tablenames := fmt.Sprintf("with t as "+
		"(select min(serial_time) startts, min(serial_time)+min(part_length)-1 endts, "+
		" tablename from %s group by 3 order by 1) "+
		"select tablename from t where "+
		" (%d between startts and endts or %d between startts and endts)"+
		" or (startts >= %d and endts <= %d) order by startts;",
		partTablename, startts, endts, startts, endts)
	var tables []string
	tables = fetchTable(q_tablenames, reflect.TypeOf(tables)).([]string)

	if len(tables) == 0 {
		randPart, ok := GetRandomPartName(partTablename)
		if !ok {
			randPart = wndName
		}
		query = fmt.Sprintf("select * from (select * from %s limit 0) foo ", randPart)
	} else {
		for i, t := range tables {
			if i != 0 {
				query += " union all "
			}
			query += fmt.Sprintf(
				"select * from %v where serial_time between %d and %d",
				t, startts, endts)
		}
	}
	return query
}

func GetPartTables(partTblname string, startts, endts int64) (parts []DbsParttable) {
	query := fmt.Sprintf("select * from %v where serial_time > %d-part_length and serial_time < %d order by serial_time asc", escape(partTblname), startts, endts)
	log.Println("GetPartTables: " + query)
	return fetchTable(query, reflect.TypeOf(parts)).([]DbsParttable)
}

func GetRandomPartName(partTblname string) (part string, ok bool) {
	query := fmt.Sprintf("select * from %v order by serial_time desc limit 1", escape(partTblname))
	log.Printf("GetRandomPartName: %v", query)
	var parts []DbsParttable
	parts = fetchTable(query, reflect.TypeOf(parts)).([]DbsParttable)
	if len(parts) == 0 {
		log.Printf("GetRandomPartName_false %v", parts)
		return "", false
	}

	log.Printf("GetRandomPartName_parts: %v", parts)
	return parts[0].Tablename, true
}

func escape(in string) string {
	in = strings.Replace(in, "'", "\\'", -1)
	in = strings.Replace(in, "\"", "\\\"", -1)
	in = strings.Replace(in, "$", "\\$", -1)
	return in
}

/*
fetchTable

Requests a table from the database and returns a slice of the given type of data.
*/
func fetchTable(query string, t reflect.Type) interface{} {
	//rows to fetch per step
	slSize := 1024

	conn, err := dbs.pool.Acquire()
	if err != nil {
		log.Fatalf("fetchTable failed.\n%v", err)
	}
	defer dbs.pool.Release(conn)


	rs, err := conn.Query(query)
	if err != nil {
		log.Fatalf("fetchTable failed.\n%v", err)
	}

	ret := reflect.MakeSlice(t, 0, 0)
	tmp := reflect.MakeSlice(t, slSize, slSize)
	var i int
	for i = 0; ; i++ {
		hasRow := rs.Next()
		if !hasRow {
			break
		}

		if i%slSize == 0 && i != 0 {
			if tmp.Index(0) != reflect.Zero(tmp.Index(0).Type()) {
				ret = reflect.AppendSlice(ret, tmp)
			}
			tmp = reflect.MakeSlice(t, slSize, slSize)
		}
		fieldCnt := 1
		if tmp.Index(i%slSize).Kind() == reflect.Struct {
			fieldCnt = tmp.Index(i % slSize).NumField()
		}

		vals, err := rs.Values()
		if err != nil {
			log.Fatalf("fetchTable scan failed.\n%v", err)
		}

		for fi := 0; fi < fieldCnt; fi++ {
			any := vals[fi]

			var field reflect.Value
			if tmp.Index(i%slSize).Kind() == reflect.Struct {
				field = tmp.Index(i % slSize).Field(fi)
			} else {
				field = tmp.Index(i % slSize)
			}

			switch v := any.(type) {
			case string:
				field.SetString(v)
			case float32:
				field.SetFloat(float64(v))
			case float64:
				field.SetFloat(v)
			case int:
				field.SetInt(int64(v))
			case int64:
				field.SetInt(int64(v))
			case bool:
				field.SetBool(v)
			}
		}
	}
	if tmp.Index(0).Interface() != nil {
		ret = reflect.AppendSlice(ret, tmp)
	}
	return ret.Slice(0, i).Interface()
}
/*
func fetchTable(query string, t reflect.Type) interface{} {
	//rows to fetch per step
	slSize := 1024

	conn, err := dbs.pool.Acquire()
	if err != nil {
		log.Fatalf("fetchTable failed.\n%v", err)
	}
	defer dbs.pool.Release(conn)

	rs, err := conn.Query(query)
	if err != nil {
		log.Fatalf("fetchTable failed.\n%v", err)
	}

	ret := reflect.MakeSlice(t, 0, 0)
	tmp := reflect.MakeSlice(t, slSize, slSize)
	var i int
	for i = 0; ; i++ {
		hasRow, err := rs.Next()
		if !hasRow {
			break
		}
		if err != nil {
			log.Fatalf("fetchTable failed.\n%v", err)
		}
		if i%slSize == 0 && i != 0 {
			if tmp.Index(0) != reflect.Zero(tmp.Index(0).Type()) {
				ret = reflect.AppendSlice(ret, tmp)
			}
			tmp = reflect.MakeSlice(t, slSize, slSize)
		}
		fieldCnt := 1
		if tmp.Index(i%slSize).Kind() == reflect.Struct {
			fieldCnt = tmp.Index(i % slSize).NumField()
		}
		for fi := 0; fi < fieldCnt; fi++ {
			any, isNull, err := rs.Any(fi)
			if err != nil {
				log.Fatalf("fetchTable scan failed.\n%v", err)
			}
			var field reflect.Value
			if tmp.Index(i%slSize).Kind() == reflect.Struct {
				field = tmp.Index(i % slSize).Field(fi)
			} else {
				field = tmp.Index(i % slSize)
			}

			if !isNull {
				switch v := any.(type) {
				case string:
					field.SetString(v)
				case float32:
					field.SetFloat(float64(v))
				case float64:
					field.SetFloat(v)
				case int:
					field.SetInt(int64(v))
				case int64:
					field.SetInt(int64(v))
				case bool:
					field.SetBool(v)
				}
			}
		}
	}
	if tmp.Index(0).Interface() != nil {
		ret = reflect.AppendSlice(ret, tmp)
	}
	return ret.Slice(0, i).Interface()
}
*/

/*
safeFetchTable

Requests a table from the database and returns a slice of the given type of data.
*/
func safeFetchTable(query string, t reflect.Type) (interface{}, error) {
	//rows to fetch per step
	slSize := 1024

	conn, err := dbs.pool.Acquire()
	if err != nil {
		return nil, err
	}
	defer dbs.pool.Release(conn)


	rs, err := conn.Query(query)
	if err != nil {
		return nil, err
	}

	ret := reflect.MakeSlice(t, 0, 0)
	tmp := reflect.MakeSlice(t, slSize, slSize)
	var i int
	for i = 0; ; i++ {
		hasRow := rs.Next()
		if !hasRow {
			break
		}

		if i%slSize == 0 && i != 0 {
			if tmp.Index(0) != reflect.Zero(tmp.Index(0).Type()) {
				ret = reflect.AppendSlice(ret, tmp)
			}
			tmp = reflect.MakeSlice(t, slSize, slSize)
		}
		fieldCnt := 1
		if tmp.Index(i%slSize).Kind() == reflect.Struct {
			fieldCnt = tmp.Index(i % slSize).NumField()
		}

		vals, err := rs.Values()
		if err != nil {
			return nil, err
		}

		for fi := 0; fi < fieldCnt; fi++ {
			any := vals[fi]

			var field reflect.Value
			if tmp.Index(i%slSize).Kind() == reflect.Struct {
				field = tmp.Index(i % slSize).Field(fi)
			} else {
				field = tmp.Index(i % slSize)
			}

			switch v := any.(type) {
			case string:
				field.SetString(v)
			case float32:
				field.SetFloat(float64(v))
			case float64:
				field.SetFloat(v)
			case int:
				field.SetInt(int64(v))
			case int64:
				field.SetInt(int64(v))
			case bool:
				field.SetBool(v)
			}
		}
	}
	if tmp.Index(0).Interface() != nil {
		ret = reflect.AppendSlice(ret, tmp)
	}
	return ret.Slice(0, i).Interface(), nil
}
/*
func safeFetchTable(query string, t reflect.Type) (interface{}, error) {
	//rows to fetch per step
	slSize := 1024

	conn, err := dbs.pool.Acquire()
	if err != nil {
		return nil, err
	}
	defer dbs.pool.Release(conn)

	rs, err := conn.Query(query)
	if err != nil {
		return nil, err
	}

	ret := reflect.MakeSlice(t, 0, 0)
	tmp := reflect.MakeSlice(t, slSize, slSize)
	var i int
	for i = 0; ; i++ {
		hasRow, err := rs.FetchNext()
		if !hasRow {
			break
		}
		if err != nil {
			return nil, err
		}
		if i%slSize == 0 && i != 0 {
			if tmp.Index(0) != reflect.Zero(tmp.Index(0).Type()) {
				ret = reflect.AppendSlice(ret, tmp)
			}
			tmp = reflect.MakeSlice(t, slSize, slSize)
		}
		fieldCnt := 1
		if tmp.Index(i%slSize).Kind() == reflect.Struct {
			fieldCnt = tmp.Index(i % slSize).NumField()
		}
		for fi := 0; fi < fieldCnt; fi++ {
			any, isNull, err := rs.Any(fi)
			if err != nil {
				return nil, err
			}
			var field reflect.Value
			if tmp.Index(i%slSize).Kind() == reflect.Struct {
				field = tmp.Index(i % slSize).Field(fi)
			} else {
				field = tmp.Index(i % slSize)
			}

			if !isNull {
				switch v := any.(type) {
				case string:
					field.SetString(v)
				case float32:
					field.SetFloat(float64(v))
				case float64:
					field.SetFloat(v)
				case int:
					field.SetInt(int64(v))
				case int64:
					field.SetInt(int64(v))
				case bool:
					field.SetBool(v)
				}
			}
		}
	}
	if tmp.Index(0).Interface() != nil {
		ret = reflect.AppendSlice(ret, tmp)
	}
	return ret.Slice(0, i).Interface(), nil
}
*/