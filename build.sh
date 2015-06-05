#!/bin/bash
# 
# Copyright (C) 2013 - 2015 - FTW Forschungszentrum Telekommunikation Wien GmbH (www.ftw.at)
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License, version 3,
# as published by the Free Software Foundation.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program. If not, see <http://www.gnu.org/licenses/>.
#
# Author(s): Arian Baer (baer _at_ ftw.at)
#
#
echo "Copyright (C) 2013 - 2015 - FTW Forschungszentrum Telekommunikation Wien GmbH (www.ftw.at)"
echo

echo "Creating directory for executables..."
set -e
if [ ! -d bin ]; then
	mkdir bin
fi
echo

PARALLEL="& "
if [ "$1" = "single" ]; then
	PARALLEL=""
fi

cd bin

echo "Building MATH importer..."
#cd ../src/modules/math_import/
#./build.sh
#cd -
set -x
MCMD1="go build ../src/modules/math_import/math_repo.go"
MCMD2="go build ../src/modules/math_import/math_probe.go"
if [ "$1" = "single" ]; then
	$MCMD1; $MCMD2;
else
	$MCMD1 & $MCMD2 &
fi
set +x
wait
echo

echo "Building DBStream main..."
set -x
CMD1="go build ../src/server/hydra.go"
CMD2="go build ../src/modules/viewgen/viewgen.go"
CMD3="go build ../src/modules/scheduler/scheduler.go"
CMD4="go build ../src/modules/external_import/external_import.go"
CMD5="go build ../src/modules/retention/retention.go"
CMD6="go build ../src/remote/remote.go"
if [ "$1" = "single" ]; then
	$CMD1; $CMD2; $CMD3; $CMD4; $CMD5; $CMD6;
else
	$CMD1 & $CMD2 & $CMD3 & $CMD4 & $CMD5 & $CMD6 &
fi

wait
set +x

cd -

echo "Creating Links"
for exec in external_import hydra math_probe math_repo remote retention scheduler viewgen
do
        ln -sf ../bin/$exec test/$exec
done

echo
echo "Build completed." 
