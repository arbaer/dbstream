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
create or replace function dbs_create_tmp_dir(name text) returns text as '
#!/bin/bash
mkdir /tmp/$1
echo /tmp/$1
' language plsh;

create or replace function dbs_rm_tmp_dir(name text) returns void as '
#!/bin/bash
rm -rf /tmp/$1
' language plsh;

create or replace function dbs_rm_tmp_dir_if_exists(name text) returns void as '
#!/bin/bash
if [ -d "/tmp/$1" ] 
then 
	rm -rf /tmp/$1
fi
' language plsh;

create or replace function dbs_exec(name text) returns void as $$
#!/bin/bash
$1
$$ language plsh;

create or replace function dbs_exec_pipe(a text, b text) returns void as $$
#!/bin/bash
$1 | $2
$$ language plsh;

create or replace function dbs_exec_pipeto(a text, b text) returns void as $$
#!/bin/bash
$1 > $2
$$ language plsh;


create or replace function dbs_exec_ret(name text) returns text as $$
#!/bin/bash
echo "`$1`"
$$ language plsh;

create or replace function dbs_weka_classify(serial_time int4, wekadir text, modelfile text, tmpdir text, inputfile text, classifier text) returns text as $$
#!/bin/bash
TMPFILE=$4/tmp.csv
OUTFILE=$4/out.csv
SERIALTIME=$1

java -cp $2/weka.jar $6 -l $2/$3 -T $4/$5 -no-cv -p first-last \
 -classifications "weka.classifiers.evaluation.output.prediction.CSV -p first -suppress -file $TMPFILE" > /dev/null
head -n -1  $TMPFILE | awk -F "," 'NR>1 {split($3,a,":"); print '$SERIALTIME' "," $6 "," a[2]}' > $OUTFILE;
echo $OUTFILE
$$ language plsh;


create or replace function dbs_weka_classify2(serial_time int4, wekadir text, modelfile text, tmpdir text, inputfile text, classes text) returns text as $$
#!/bin/bash
INFILE=$4/$5
INARFF=$4/$5.arff
TMPARFF=$4/tmp.arff
TMPARFF_CLASS=$4/tmp_class.arff
TMPOUTFILE=$4/tmp_out.csv
OUTFILE=$4/out.csv
SERIALTIME=$1

java -cp $2/weka.jar weka.core.converters.CSVLoader -S 1 -L last:$6 $INFILE > $INARFF
java -classpath $2/weka.jar weka.filters.supervised.attribute.AddClassification -serialized $2/$3 -classification  -i $INARFF -o $TMPARFF -c last \
	3>&1 1>&3 2>&1 | awk '{if (length($0)> 0 && !($0 ~ /GConf-WARNING/ || $0 ~ /GConf Error/ || $0 ~ /Unable to autolaunch/ || $0 ~ /dbus-launch terminated/ || $0 ~ /X11 initialization failed/))  print $0}' 1>&2 3>&1
java -classpath $2/weka.jar weka.filters.unsupervised.attribute.Remove -R first,last -V -i $TMPARFF -o $TMPARFF_CLASS \
	3>&1 1>&3 2>&1 | awk '{if (length($0)> 0 && !($0 ~ /GConf-WARNING/ || $0 ~ /GConf Error/ || $0 ~ /Unable to autolaunch/ || $0 ~ /dbus-launch terminated/ || $0 ~ /X11 initialization failed/))  print $0}' 1>&2 3>&1
java -classpath $2/weka.jar weka.core.converters.CSVSaver -N -i $TMPARFF_CLASS -o $TMPOUTFILE \
	3>&1 1>&3 2>&1 | awk '{if (length($0)> 0 && !($0 ~ /GConf-WARNING/ || $0 ~ /GConf Error/ || $0 ~ /Unable to autolaunch/ || $0 ~ /dbus-launch terminated/ || $0 ~ /X11 initialization failed/))  print $0}' 1>&2 3>&1
awk -F "," '{print "'$SERIALTIME',"$0}' $TMPOUTFILE > $OUTFILE
echo $OUTFILE

$$ language plsh;

create or replace function dbs_weka_classify_with_model(serial_time int4, tablename text, modelfile text, columns text, classes text) returns table (serial_time_out int4, id text, class text) as $$
declare 
	wekadir text;
	dirname text;
	message text;
	weka_cmd text;
	importfile text;
	cnt int8;
	rand int4;
begin

--classifier e.g. weka.classifiers.trees.HoeffdingTree

wekadir = '/home/metawin/weka';

select (RANDOM()*1000000)::int4 into rand;

select dbs_create_tmp_dir(modelfile||'_'||rand) into dirname;
perform dbs_rm_tmp_dir_if_exists(dirname);
raise notice 'dirname:%', dirname;

create temp table imp (serial_time_out int4, id text, class text) on commit drop;

execute 'select count(*) from (select * from '||tablename||' limit 1) a;' into cnt;
if cnt > 0 then
	execute 'copy (select '||columns||' from '||tablename||E') to \''||dirname||E'/learn.csv\' (format csv, header true, NULL \'?\')';

	begin
		perform dbs_weka_classify2(serial_time, wekadir, modelfile, dirname, '/learn.csv', classes);
	exception
		when OTHERS then
		GET STACKED DIAGNOSTICS message = MESSAGE_TEXT;
		raise exception '%', message;
	end;
	
	importfile = dirname||'/out.csv';
	
	raise notice 'importfile:%', importfile;
		
	execute E'copy imp from \''||importfile||E'\' (format csv)';
	
	perform dbs_rm_tmp_dir_if_exists(dirname);		
end if;

return query select * from imp;
	

return;


EXCEPTION
	when OTHERS then 
	GET STACKED DIAGNOSTICS message = MESSAGE_TEXT;
	raise exception '%', message;
	perform dbs_rm_tmp_dir_if_exists(dirname);	


end;
$$ language plpgsql;
