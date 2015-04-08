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

cd bin

echo "Building MATH importer..."
#cd ../src/modules/math_import/
#./build.sh
#cd -
set -x
go build ../src/modules/math_import/math_repo.go &
go build ../src/modules/math_import/math_probe.go &
set +x
wait
echo

echo "Building DBStream main..."
set -x
go build ../src/server/hydra.go &
go build ../src/modules/viewgen/viewgen.go &
go build ../src/modules/scheduler/scheduler.go &
go build ../src/modules/external_import/external_import.go &
go build ../src/modules/retention/retention.go &
go build ../src/remote/remote.go &
wait
set +x

cd -

echo
echo "Build completed." 