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
# Author(s): Arian Baer (baer _at_ ftw.at), Thomas Paulin (paulin _at_ ftw.at)
#
#

import matplotlib.ticker as ticker
import matplotlib.cm as cm
import time, calendar, datetime, sys
import numpy as np
import operator
from matplotlib import pyplot as plt


class ScatterPlotter:

	def __init__(self):
		self.hatches = ['/', 'o', '|', '\\','+', 'x', '*',  'O', '.', '-']

		return

	def format_time(self, x, pos=None):
		time = datetime.datetime.fromtimestamp(x)
		ret = str(time).replace(' ', '\n')
		return ret

	def format_tiny(self, x, pos=None):
		if x >= 1000000:
			ret = str(float(x)/1000000) + "M"
		elif x >= 10000:
			ret = str(int(x)/1000) + "K"
		elif x >= 1000:
			ret = str(float(x)/1000) + "K"
		else:
			ret = str(int(x))
		return ret

	def format_date(self, x, pos=None):
		time = datetime.datetime.fromtimestamp(x)

		ret = str(time.year) + "-" + str(time.month) + "-" + str(time.day)
		return ret

	def format_days(self, x, pos=None):
		return int((x-(x%86400) - self.start_day)/86400)

	def format_dict(self, x, pos=None):
		ret = ""
		if self.xFormDict.has_key(str(int(x))):
			ret = self.xFormDict[str(int(x))]
		else:
			ret = str(int(x)) 
		return ret


	def plot_single(self, fig, datafile, line_style):
		f = open(datafile)

		x = list()
		y = list()

		for line in f:
			line = line.split('\t')
			x.append(line[0])
			y.append(line[1])
		f.close()

		ax = fig.add_subplot(111)
		ax.plot(x, y, line_style)
		ax.grid(True)
		return ax


	def calc_cdf(self, vals):
		total_vals = 0.0
		for v in vals:
			total_vals += float(v) if '.' in v else int(v)

		print total_vals
		nu_vals = list()
		nu_val = 0.0
		nu_vals.append(0)
		for v in vals:
			nu_val += float(v) if '.' in v else int(v)
			nu_vals.append(nu_val/total_vals)

		return nu_vals

	def calc_ccdf(self, vals):
		total_vals = 0.0
		for v in vals:
			total_vals += float(v) if '.' in v else int(v)

		print total_vals
		nu_vals = list()
		nu_val = 0.0
		for v in vals:
			nu_vals.append(1 - nu_val/total_vals)
			nu_val += float(v) if '.' in v else int(v)
			nu_vals.append(1 - nu_val/total_vals)

		return nu_vals


	def ecdf(self, x, y):
		total_vals = 0.0
		ys = [ float(v) if '.' in v else int(v) for v in y ]
		xs = [ float(v) if '.' in v else int(v) for v in x ]

		y_vals = list()
		x_vals = list()

		for i in np.arange(0,len(xs)):
			if i < len(xs) - 1:
				x_vals.append(xs[i])
				x_vals.append(xs[i+1])
			else:
				x_vals.append(xs[i])
				x_vals.append(xs[i])


		total_vals = sum(ys)

		nu_val = 0.0
		for y in ys:
			y_vals.append(1 - nu_val/total_vals)
			y_vals.append(1 - nu_val/total_vals)

			nu_val += y

		return x_vals, y_vals



	def plot_multi(self, fig, datafile, line_style, xformat, legendStyle, sort, loc, markeveryInp):
		f = open(datafile)

		hasErrorBar = False

		hash = dict()

		sub_plot_num = 111
		for line in f:
			line = line.split('\t')

			if(len(line) > 3):
				sub_plot_num = int(line[3].strip())

			p_name = line[2].strip()
			#if not hash.has_key(sub_plot_num):
			if not sub_plot_num in hash: # Does not work in Python3
				hash[sub_plot_num] = dict()
			if not p_name in hash[sub_plot_num]:
				x = list()
				y = list()
				err = list()
				hash[sub_plot_num][p_name] = (x, y, err)

			hash[sub_plot_num][p_name][0].append(line[0])

			if len(line[1].split(';')) == 1: # Check for error bars
				hash[sub_plot_num][p_name][1].append(line[1])
				hash[sub_plot_num][p_name][2].append(0) # 

			else:
				hasErrorBar = True
				hash[sub_plot_num][p_name][1].append(line[1].split(';')[0]) # Append Y value
				hash[sub_plot_num][p_name][2].append(line[1].split(';')[1]) # Append Error Value


		f.close()

		subplots = list()
		ax1 = 0
		ticks = list()
		sorted_hash = sorted( iter( hash.keys() )  )
		for sub_plot_num in sorted_hash:
			if(ax1 == 0):
				ax = fig.add_subplot(sub_plot_num)
				ax1 = ax
			else:
				ax = fig.add_subplot(sub_plot_num, sharex=ax1)
			subplots.append(ax)
			plots = list()
			if sort == "int":
				sorted_names = map(lambda x:str(x), sorted(map(lambda x:int(x),hash[sub_plot_num])))
			else:
				sorted_names = sorted(hash[sub_plot_num])
#			for p_name in hash[sub_plot_num]:
			i = 0
			cm_len = float(len(sorted_names))
			y_valsStacked = list()
			longestStacked = 0
			for p_name in sorted_names:
				if xformat == 'cdf':
					y_vals = self.calc_cdf(hash[sub_plot_num][p_name][1])
					x_vals = hash[sub_plot_num][p_name][0]
					hash[sub_plot_num][p_name][0][:0] = [0] #prepend 0
				elif xformat == 'ccdf':
					y_vals = self.calc_ccdf(hash[sub_plot_num][p_name][1])
					x_vals = hash[sub_plot_num][p_name][0]
					#hash[sub_plot_num][p_name][0].append(x_vals[-1]) 
					#x_vals = hash[sub_plot_num][p_name][0]
				elif xformat == 'eccdf': # ECCDF has to update the X values as well.
					x_vals, y_vals = self.ecdf( hash[sub_plot_num][p_name][0], hash[sub_plot_num][p_name][1] )
				else:
					if hasErrorBar:
						y_vals = [float(a) for a in hash[sub_plot_num][p_name][1]]
						y_err  = [float(a) for a in hash[sub_plot_num][p_name][2]]

						y_err_high = [ float(a) + float(b) for a,b  in zip(hash[sub_plot_num][p_name][1], hash[sub_plot_num][p_name][2]) ]
						y_err_low  = [ float(a) - float(b) for a,b  in zip(hash[sub_plot_num][p_name][1], hash[sub_plot_num][p_name][2]) ]
					else:

						y_vals = hash[sub_plot_num][p_name][1]
					x_vals = hash[sub_plot_num][p_name][0]


				if line_style == 'multi':
					dots = ".x+"
					ax.plot(x_vals, y_vals, dots[i%len(dots)], c=cm.hsv(i/cm_len,1))
				elif line_style == 'multi-':
					dots = ["-v","-x","-o"]
					if hasErrorBar:
						plt.errorbar(x_vals, y_vals, fmt=dots[i%len(dots)], c=cm.hsv(i/cm_len,1), yerr=y_err)
					else:
						ax.plot(x_vals, y_vals, dots[i%len(dots)], c=cm.hsv(i/cm_len,1))
				elif line_style == 'multi--':
					dots = ["--v","--x","--o"]
					if hasErrorBar:
						#plt.errorbar(x_vals, y_vals, fmt=dots[i%len(dots)], c=cm.gnuplot(i/cm_len,1), yerr=y_err)
						plt.errorbar(x_vals, y_vals, fmt=dots[i%len(dots)], yerr=y_err)
					else:
						ax.plot(x_vals, y_vals, dots[i%len(dots)], c=cm.gnuplot(i/cm_len,1), )

				elif line_style == 'multismall-':
					dots = ["-","-+","-x"]
					ax.plot(x_vals, y_vals, dots[i%len(dots)], c=cm.hsv(i/cm_len,1))
				elif line_style == 'multismall':
					dots = [".","+","x"]
					ax.plot(x_vals, y_vals, dots[i%len(dots)], c=cm.hsv(i/cm_len,1))
				elif line_style == 'bnw':
					dots = ["h", "+", "x", "d", "v", "o", "D", "."]
					ax.plot(x_vals, y_vals, dots[i%len(dots)])
				elif line_style == 'bnw-':
					dots = ["-h", "-+", "-x", "-d", "-v", "-o", "-D", "-."]
					ax.plot(x_vals, y_vals, dots[i%len(dots)], markevery=int(markeveryInp))
					#ax.plot(x_vals, y_vals, dots[i%len(dots)])
				elif xformat == 'stack':
					y = np.array(y_vals, dtype=float)
					y_valsStacked.append(y)
					if longestStacked < len(y): 
						longestX = x_vals
					longestStacked = max(longestStacked, len(y))
				elif xformat == 'fill':
					ax.fill(x_vals, y_vals, line_style, edgecolor='black', hatch=self.hatches[i%len(self.hatches)], color=cm.Pastel2(i/cm_len,1))
				else:
					if hasErrorBar:	
						plt.errorbar(x_vals, y_vals, linestyle=line_style, yerr=y_err)
					
					else:
						ax.plot(x_vals, y_vals, line_style)
					
				plots.append(p_name)
				i += 1


			if xformat == 'stack':
				print longestStacked
				#ys = [np.array(y, dtype=float).resize(longestStacked) for y in y_valsStacked]
				stacks = list()
				y_cnt = 0
				for y in y_valsStacked:
					y = np.copy(y)
					y.resize(longestStacked)
					print y
					stacks.append(y)
					y_cnt +=1 
#				ys = [np.array(y, dtype=float) for y in y_valsStacked]
#				ys = [y.resize(longestStacked) for y in ys]
				ax.stackplot(np.array(longestX, dtype=float), *stacks, baseline='wiggle')

			if legendStyle == 'none':
				pass
			elif (legendStyle == 'single' or legendStyle == 'single-outside') and i > 1:
				pass
			elif legendStyle == 'outside':
				leg = ax.legend(plots, bbox_to_anchor=(1.01, 1), loc=2, borderaxespad=0.5)
			elif legendStyle == 'below':
				ax.legend(plots, loc='upper center', bbox_to_anchor=(0.5, -0.15), fancybox=True, shadow=True, ncol=2)
			else:
				try:
					loc = int(loc)
				except:
					loc = loc
				finally:
					leg = ax.legend(plots, loc=loc)

			ax.grid(True)

		if legendStyle == 'single':
			leg = subplots[0].legend(plots, 'best')
		elif legendStyle == 'single-outside':
			leg = subplots[0].legend(plots, bbox_to_anchor=(1.01, 1), loc=2, borderaxespad=0.5)


		if xformat == 'time':
			ax.xaxis.set_major_formatter(ticker.FuncFormatter(self.format_time))
			fig.autofmt_xdate()
		elif xformat == 'date':
			ax.xaxis.set_major_formatter(ticker.FuncFormatter(self.format_date))
			fig.autofmt_xdate()
		elif  xformat == 'days':
			min_time = int(min(x))
			self.start_day = min_time - min_time%86400
			ax.xaxis.set_major_formatter(ticker.FuncFormatter(self.format_days))
		elif  xformat == 'tiny':
			ax.xaxis.set_major_formatter(ticker.FuncFormatter(self.format_tiny))
			ax.yaxis.set_major_formatter(ticker.FuncFormatter(self.format_tiny))
			#fig.autofmt_xdate()
		elif xformat == 'dict':
			ax.xaxis.set_major_formatter(ticker.FuncFormatter(self.format_dict))
			fig.autofmt_xdate()

		return subplots

	def plot_bar(self, fig, datafile, xnames):
		  f = open(datafile)
		  #f = open("plot_exec_time.txt")

		  barGroups = dict()
		  locations = list()

		  barNameOrder = list()

		  xnameLookup = dict()

		  hasErr = False

		  confSplit = ';'

		  for line in f:
			  if line[0] == '#':
				  continue
			  (x,y,text) = line.split('\t')
			  yval = float(y.split(confSplit)[0])
			  yerr = 0.0

			  if confSplit in y:
				  hasErr = True
				  s = y.split(confSplit)
				  yerr = float(s[1])

			  text = text.strip()
			  if text not in barNameOrder:
				  barNameOrder.append(text)
			  if text not in barGroups:
				  barGroups[text] = dict()
			  if x not in barGroups[text]:
				  barGroups[text][x] = dict()
			
			  barGroups[text][x]['bar'] = yval
			  barGroups[text][x]['err'] = yerr

			  if x not in xnameLookup:
				  i = len(locations)
				  locations.append(i)
				  xnameLookup[x] = i

		  width = 0.5/len(barGroups)

		  locations = sorted(locations)

		  loc = np.arange(len(locations))
		  pos = 0
		  for l in locations:
			  loc[pos] = int(l)
			  pos += 1

		  ax = fig.add_subplot(111)

		  numPlots = 0
		  barLen = float(len(barGroups))
		  rects = list()
		  names = list()

			
		  cur_hatch = 0
		  for b in barNameOrder:
			  print b
			  values = barGroups[b]

			  barPlot = list()
			  barErrPlot = list()

			  for l in locations:
				  found = False
				  for k in xnameLookup:
					  if xnameLookup[k] == l and k in values:
#				  if l in values and values[l] > 0:
						  barPlot.append(values[k]['bar'])
						  barErrPlot.append(values[k]['err'])
						  found = True
				  if not found:
					  barPlot.append(0.0)
					  barErrPlot.append(0.0)

			  ax.set_yscale('linear')
			  if hasErr:
				  rect = ax.bar(loc+(numPlots*width), barPlot, width, color=cm.Pastel2(numPlots/barLen,1), hatch=self.hatches[cur_hatch%(len(self.hatches))], yerr=barErrPlot, ecolor='black', capsize=5)
			  else:
				  rect = ax.bar(loc+(numPlots*width), barPlot, width, color=cm.Pastel2(numPlots/barLen,1), hatch=self.hatches[cur_hatch%(len(self.hatches))])
			  cur_hatch += 1


			  rects.append(rect[0])
			  names.append(b)
			  ax.grid(which='major', color='0.65',linestyle='--')

			  numPlots += 1
			  #[line.set_zorder(3) for line in ax.lines]
			  ax.set_axisbelow(True)

		  ax.set_ylabel('Execution Time [minutes]')
		  ax.set_xticks(loc+0.2)
		  
		  ax.yaxis.set_major_formatter(ticker.FuncFormatter(self.format_tiny))

		  labels = sorted(xnameLookup, key=lambda x: xnameLookup[x])
		  ax.set_xticklabels( (labels), rotation=0 )


		  ax.legend(rects, names, 'best')
		  return [ax]

	def plot(self, fig, datafile, line_style, xformat, legendStyle, xFormDict, sort, xnames, loc, markevery):
		f = open(datafile)
		self.xFormDict = xFormDict
		test_line = f.readline()
		f.close()
		if len(test_line.split('\t')) == 2:
			print "plot single"
			return self.plot_single(fig, datafile, line_style)
		elif xformat == "bar":
			return self.plot_bar(fig, datafile, xnames)
		if xformat == "bar":
			return self.plot_bar(fig, datafile, xnames)
		else:
			print "plot multi"
			return self.plot_multi(fig, datafile, line_style, xformat, legendStyle, sort, loc, markevery)

