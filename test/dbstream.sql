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
create group dbs_test;
alter user dbs_test login;

create schema dbs;

create schema dbs_ts0;
create tablespace dbs_ts0 owner dbs_test location '/home/dbs_test/dbs_ts0';

CREATE TABLE dbs_tables (
	tablename character varying(256),
	part_tablename character varying(256),
	tabletype character varying(256)
);

create table dbs.query_stats (
	timestamp timestamp,
	job_series text,
	job text,
	inputs text,
	output text,
	input_time int8,
	exec_time float,
	affected_rows int8,
	additional text
);

