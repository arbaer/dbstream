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
create or replace function dbs_time(ts text) returns int4 as $$
declare
ret int4;
begin
	select extract(epoch from ts::timestamp with time zone)::int4 into ret;
	return ret;
end;
$$ language plpgsql strict immutable;


create or replace function dbs_create_table(tblname text, schema text) returns void as $$
declare
	part_tblname text;
begin
	execute 'create table '||tblname||' ('||schema||')';
	part_tblname = 'view_info_'||tblname;
	execute 'create table dbs.'||part_tblname||' (serial_time int4, part_length int4, hostname text, port int4, dbname text, tablename text)';
	execute 'create index '||part_tblname||'_serial_time_idx on dbs.'||part_tblname||' (serial_time)';
	execute 'insert into dbs_tables values ('''||tblname||''', ''dbs.'||part_tblname||''', ''basetable'')';
end;
$$ strict language plpgsql;


create or replace function dbs_drop_table(tblname text) returns void as $$
begin
	execute 'drop table if exists '||tblname||' cascade';
	execute 'drop table if exists dbs.view_info_'||tblname;
	execute 'delete from dbs_tables where tablename = '''||tblname||'''';
end;
$$ strict language plpgsql;


create or replace function dbs_alter_schema_to(tblname text, newSchema text) returns void as $$
begin
	execute 'alter table '||tblname||' set schema '||newSchema;
	raise notice 'changed schema of table % to %', tblname, newSchema;
end;
$$ strict language plpgsql;


create or replace function dbs_alter_table(tblname text, alterStr text) returns void as $$
declare
	q text;
begin

	q = 'alter table '||tblname||' '||alterStr;
	execute q;
	raise notice 'executed: %', q;
end;
$$ strict language plpgsql;

create or replace function dbs_truncate_table(tblname text) returns void as $$
begin
	execute 'drop table '||tblname||' cascade';
	execute 'drop table dbs.view_info_'||tblname;
	execute 'delete from dbs_tables where tablename = '''||tblname||'''';
end;
$$ strict language plpgsql;


create or replace function dbs_delete_from_table(tblname text, startts int4) returns void as $$
begin
	perform dbs_delete_from_table(tblname, startts, x'7fffffff'::int4);
end;
$$ strict language plpgsql;


create or replace function dbs_delete_from_table(tblname text, startts int4, endts int4) returns void as $$
declare
	tbl record;
	start_tbl text;
	end_tbl text;
	query text;
	max_serial_time int4;
	sub_part_len int4;
begin
	if endts = -1 then
		endts = x'7fffffff'::int4;
	end if;

	--delete data from the starting partition
	query = 'with t as (select min(serial_time) serial_time, part_length, tablename from dbs.view_info_'||tblname||' group by 2,3 order by 1) '||
		'select tablename from t where serial_time < '||startts||' and serial_time+part_length > '||startts;
	raise warning '%', query;
	execute query into start_tbl;
	if start_tbl is not null then
		query = 'delete from '||start_tbl||' where serial_time >= '||startts;
		raise warning '%', query;
		execute query;
	end if;

	--delete intermediate partitions completely
	for tbl in execute 'with t as (select min(serial_time) serial_time, part_length, tablename from dbs.view_info_'||tblname||' group by 2,3 order by 1) '||
			'select tablename from t where serial_time >= '||startts||' and serial_time+part_length-1 < '||endts||' order by serial_time asc' loop
		query = 'drop table if exists '||tbl.tablename||' cascade';
		raise warning '%', query;
		execute query;
	end loop;
	--delete data from the last partition, if any
	execute 'select max(serial_time) from dbs.view_info_'||tblname into max_serial_time;
	if max_serial_time >= endts then

		execute 'with t as (select min(serial_time) serial_time, part_length, tablename from dbs.view_info_'||tblname||' group by 2,3 order by 1) '||
			'select tablename from t where serial_time < '||endts||' and serial_time+part_length > '||endts into end_tbl;
		if end_tbl is not null then
			query = 'delete from '||end_tbl||' where serial_time < '||endts;
			raise warning '%', query;
		end if;
		execute query;

	end if;
	--delete data from the view_info
	execute 'select part_length/count(*) sub_part_len from dbs.view_info_'||tblname||' group by tablename, part_length limit 1' into sub_part_len;
	query = 'delete from dbs.view_info_'||tblname||' where serial_time >= '||startts||' and serial_time < '||endts-(endts%sub_part_len);
	raise warning '%', query;
	execute query;
end;
$$ strict language plpgsql;


create or replace function dbs_table_availability(tblname text) returns table (serial_time int4, availability int4) as $func$
declare
	mints int4;
	maxts int4;
	part_length int4;
	part_tblname text;
begin
	select part_tablename from dbs_tables where tablename=tblname into part_tblname;
	execute 'select min(serial_time)  from '||part_tblname into mints;
	execute 'select max(serial_time)  from '||part_tblname into maxts;
	execute 'select distinct part_length from '||part_tblname into part_length;

	raise warning '%', part_tblname;
	return query execute 'select a serial_time, 0 availability from generate_series('||mints||', '||maxts||', '||part_length||') a where not exists (select 1 from '||part_tblname||' b where a=b.serial_time) union all select serial_time, 1 from '||part_tblname||' order by 1';
end;
$func$ strict immutable language plpgsql;


create or replace function dbs_delete_old_days(tblname text, days int4) returns void as $$
declare
	part_tblname text;
	min_ts int4;
	min_date timestamp with time zone;
	max_date timestamp with time zone;
	tbl text;
	where_cl text;
begin
	select part_tablename from dbs_tables where tablename=tblname into part_tblname;
	raise warning 'part_tblname %', part_tblname;
	execute 'select min(serial_time) from '||part_tblname into min_ts;
	min_date = date_trunc('day', to_timestamp(min_ts));
	max_date = min_date + (days||' days')::interval;

	raise warning 'droping tables from % to %', min_date, max_date;

	where_cl := 'from '||part_tblname||' where serial_time >= '||extract(epoch from min_date)||' and serial_time < '||extract(epoch from max_date);
	for tbl in execute
		'select tablename '||where_cl||' order by serial_time' loop
		execute 'drop table if exists '||tbl;
		raise warning 'tbl % deleted.', tbl;
	end loop;
	execute 'delete '||where_cl;
end;
$$ strict language plpgsql;


create or replace function dbs_table_info(tbl text) returns 
table (
 	serial_time_ts
	timestamp with time zone,
	serial_time    integer, 
	tablename      character varying(256),
	pg_size_pretty text   
) as $$
begin
	return query execute 'select to_timestamp(serial_time), serial_time, tablename, pg_size_pretty(pg_total_relation_size(tablename)) from dbs.view_info_'||tbl;
end;
$$ strict immutable language plpgsql;


create or replace function dbs_table_stats(tbl text) returns table (tablename text, tbl_size text, avg_size_per_day text, total_tbl_size text, total_avg_size_per_day text, retention_in_days interval, start_ts timestamp with time zone, end_ts timestamp with time zone) as $$
declare
	days int4;
	min_st int4;
	max_st int4;
	rel_size int8;
	tot_rel_size int8;
	partTbl text;
begin
	select part_tablename from dbs_tables where dbs_tables.tablename=tbl into partTbl;
	execute 'select min(serial_time) from '||partTbl into min_st;
	execute 'select max(serial_time) from '||partTbl into max_st;

	select extract(days from to_timestamp(max_st)-to_timestamp(min_st)) into days;
	if days = 0 then
		days = 1;
	end if;

	return query execute 'select '
	||''''||tbl||'''::text, '
	||'pg_size_pretty(sum(pg_relation_size(tablename))::int8), '
	||'pg_size_pretty((sum(pg_relation_size(tablename))/'||days||')::int8), '
	||'pg_size_pretty(sum(pg_total_relation_size(tablename))::int8), '
	||'pg_size_pretty((sum(pg_total_relation_size(tablename))/'||days||')::int8) as avg_size_per_day, '
	||'to_timestamp(max(max_serial_time))-to_timestamp(min(min_serial_time)), '
	||'to_timestamp(min(min_serial_time)) as start_ts, '
	||'to_timestamp(max(max_serial_time)) as end_ts '
	||'from (select min(serial_time) as min_serial_time, max(serial_time) as max_serial_time, tablename from '||partTbl||' group by tablename) foo';
end;
$$ strict immutable language plpgsql;


create or replace function dbs_plot(query text, columns text) 
returns table(serial_time int4, value numeric, line_name text) as $func$
declare
col text;

ret_q text;
begin

	ret_q = '';
	for col in select regexp_split_to_table(columns, E'\\s*,\\s*') loop
	if length(ret_q) > 0 then
		ret_q = ret_q || ' union all ';
	end if;
	ret_q = ret_q || ' select serial_time, coalesce('||col||', 0.0)::numeric, '''||col||'''::text from ('||query||') as '||col||'_t ';
	end loop;
	ret_q = ret_q || ' order by serial_time';
	raise warning 'query: %', ret_q;
	return query execute ret_q;
end;
$func$ language plpgsql strict;
