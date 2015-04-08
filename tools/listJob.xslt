<!--/* 
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
*/-->
<?xml version="1.0"?>

<xsl:stylesheet version="1.0"
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

	<xsl:output method="html" version="4.0"
			encoding="iso-8859-1" indent="yes"/>


	<xsl:template match="job">
		<tr>
			<td><b><xsl:value-of select="substring-before(@output, ' ')" /></b></td>
			<td><xsl:value-of select="@description" /></td>
			<td><xsl:value-of select="@output" /></td>
			<td><xsl:value-of select="@inputs" /></td>
			<td><xsl:value-of select="@schema" /></td>
		</tr>
	</xsl:template>

	<xsl:template match="/">
		<html>
			<head>
				<style type="text/css">
				table, th, td {
					border-collapse: collapse;
					border: 1px solid black;
				}
				tr:nth-child(2n) {
					background-color: rgb(255, 255, 255);
				}
				tr:nth-child(2n+1) {
					background-color: rgb(240, 240, 240);
				}
				</style>
			</head>
			<body>
				<table>
					<thead>
						<tr>
							<th>Name</th>
							<th style="width:30%">Description</th>
							<th>Output</th>
							<th>Input</th>
							<th style="width:20%">Schema</th>
						</tr>
					</thead>
					<xsl:apply-templates select="*"/>
				</table>
			</body>
		</html>
	</xsl:template>

	<xsl:template match="*">
		<xsl:apply-templates select="*"/>
	</xsl:template>


</xsl:stylesheet>


