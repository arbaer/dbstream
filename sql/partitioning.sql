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
create or replace function dbs_create_view(name text, basetbl text, startts int4, endts int4) returns void as $$
declare
	q text;
	q_view text;
	first boolean;
	tbl record;
begin
	first = true;
	q = 'create view '||name||' as (';
	q_view = '';
	for tbl in execute 'select distinct tablename from dbs.view_info_'||basetbl||' where serial_time >='||startts||' and serial_time <'||endts loop
		if first then
			first = false;
		else
			q_view = ' union all '||q_view;
		end if;
		q_view = 'select * from '||tbl.tablename||' '||q_view;
	end loop;
	q = q ||q_view||');';
	raise warning '%', q;
	execute q;
end;
$$ language plpgsql strict;
