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
--staleness

copy (select rtime, (rtime-vtime)/(3600.0), job_series from (select extract(epoch from timestamp) rtime, timestamp as rtime_date, to_timestamp(input_time) 
		vtime_date, input_time as vtime, job_series from dbs.query_stats where job_series not like 'test%') foo order by 1) to '/tmp/staleness.txt';

--query performance
copy (select input_time, exec_time, job_series from dbs.query_stats where job_series not like 'test%' and exec_time is not null order by 1) to '/tmp/exec_time_input_time.txt';
copy (select extract(epoch from timestamp), exec_time, job_series from dbs.query_stats where job_series not like 'test%' and exec_time is not null order by 1) to '/tmp/exec_time_query_time.txt';

