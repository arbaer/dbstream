#!/usr/bin/python
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
try:
	import gtk
	useGTK = True
except:
	print ("Could not load gtk, faling back to pure matplotlib")
	useGTK = False
import sys
import os
import matplotlib.pyplot as plt
import matplotlib
from matplotlib import rc
import numpy as np
from scatter_plotter import ScatterPlotter as scplt
#from time_plotter import TimePlotter as tplt

if useGTK:
	from matplotlib.backends.backend_gtkagg import FigureCanvasGTKAgg as FigureCanvas
	from matplotlib.backends.backend_gtkagg import NavigationToolbar2GTKAgg as NavigationToolbar

from optparse import OptionParser

class Plotter:

	def cmd_xlog_handler(self, widget, data=None):
		ax = data[0][0]
		canvas = data[1]
		if ax.get_xscale() == 'log':
			for plot in data[0]:
				ax.set_xscale('linear')
				plot.set_xlabel(self.xlabel)
		else:
			for plot in data[0]:
				plot.set_xscale('log')
				plot.set_xlabel(self.xlabel + " (log scale)")
		canvas.draw()
		return
	def cmd_ylog_handler(self, widget, data=None):
		ax = data[0][0]
		canvas = data[1]
		if ax.get_yscale() == 'log':
			plotnum = 0
			for plot in data[0]:
				plot.set_yscale('linear')
				plot.set_ylabel(self.ylabel[plotnum])
				plotnum += 1
		else:
			ax.set_yscale('log')
			plotnum = 0
			for plot in data[0]:
				plot.set_yscale('log')
				plot.set_ylabel(self.ylabel[plotnum] + " (log scale)")
				plotnum += 1
		canvas.draw()
		return

	def entry_fontSize_changed(self, widget, data=None):
		entry_fontSize = data[0]
		canvas = data[1]
		size = entry_fontSize.get_text()
		try:
			sizeInt = int(size)
			matplotlib.rcParams.update({'font.size':sizeInt})
		except:
			print "entered value must be a integer, got: "+size
			pass
		canvas.draw()
		return


	def dictFromFile(self, name):
		ret = dict()
		f = open(name)
		for line in f:
			k,v = line.split('\t')
			ret[k] = v
		return ret

	def doSaveToFile(self, fig):
		import PlotterHelper as ph
		ph.UpdatePlot(	fig, withLegendFrame=False)
		ph.PrintFigure(	fig, self.saveToFileName, scale=True ) 

	def __init__(self, options, args):
		
		self.title = dict()
		self.title[0] =  "use -t to add a title"
		if hasattr(options, "title") and options.title:
			self.title[0] = options.title
		if hasattr(options, "title1") and options.title1:
			self.title[1] = options.title1
		if hasattr(options, "title2") and options.title2:
			self.title[2] = options.title2
		if hasattr(options, "title3") and options.title3:
			self.title[3] = options.title3
		if hasattr(options, "title4") and options.title4:
			self.title[4] = options.title4
		self.xlabel =  "use -x to add a xlabel"
		if hasattr(options, "xlabel") and options.xlabel:
			self.xlabel = options.xlabel
		self.ylabel = dict()
		self.ylabel[0] = "use -y to add a ylabel"
		if hasattr(options, "ylabel") and options.ylabel:
			self.ylabel[0] = options.ylabel
			self.ylabel[1] = options.ylabel
			self.ylabel[2] = options.ylabel
			self.ylabel[3] = options.ylabel
			self.ylabel[4] = options.ylabel
		if hasattr(options, "ylabel1") and options.ylabel1:
			self.ylabel[1] = options.ylabel1
		if hasattr(options, "ylabel2") and options.ylabel2:
			self.ylabel[2] = options.ylabel2
 		if hasattr(options, "ylabel3") and options.ylabel3:
			self.ylabel[3] = options.ylabel3
		if hasattr(options, "ylabel4") and options.ylabel4:
			self.ylabel[4] = options.ylabel4
		self.args = args

		self.plot_type = 'scatter'
		if hasattr(options, "plot_type") and options.plot_type:
			self.plot_type = options.plot_type
		if hasattr(options, "line_style") and options.line_style:
			self.line_style = options.line_style
		else:
			self.line_style = '.'
		if hasattr(options, "ymin") and options.ymin:
			self.ymin = float(options.ymin)
		if hasattr(options, "y1min") and options.y1min:
			self.y1min = float(options.y1min)
		if hasattr(options, "ymax") and options.ymax:
			self.ymax = float(options.ymax)
		if hasattr(options, "y1max") and options.y1max:
			self.y1max = float(options.y1max)
		if hasattr(options, "xmin") and options.xmin:
			self.xmin = float(options.xmin)
		if hasattr(options, "xmax") and options.xmax:
			self.xmax = float(options.xmax)
		if hasattr(options, "xticks") and options.xticks:
			self.xticks = float(options.xticks)
		if hasattr(options, "yticks") and options.yticks:
			self.yticks = float(options.yticks)
		self.xnames = list()
		if hasattr(options, "xnames") and options.xnames:
			self.xnames = options.xnames
		self.legendStyle = ""
		if hasattr(options, "legendStyle") and options.legendStyle:
			self.legendStyle = options.legendStyle
		self.fontSize = 14
		if hasattr(options, "fontSize") and options.fontSize:
			self.fontSize = options.fontSize
		if hasattr(options, "xdict") and options.xdict:
			self.xFormDict = self.dictFromFile(options.xdict)
		else:
			self.xFormDict = dict()
		if hasattr(options, "sort") and options.sort:
			self.sort = options.sort
		else:
			self.sort = "string"
		markevery=1
		if hasattr(options, "markevery") and options.markevery:
			self.markevery = options.markevery
		else:
			self.markevery = 1
		if hasattr(options, "filename") and options.filename:
			self.saveToFile = True
			self.saveToFileName = options.filename
		else:
			self.saveToFile = False
		if hasattr(options, "loc") and options.loc:
			self.legendLoc = options.loc
		else:
			self.legendLoc = 'best'
		
		return


	def show(self):

		if useGTK:
			self.win = gtk.Window()
			self.win.connect("destroy", lambda x: gtk.main_quit())
			self.win.set_default_size(700,500)
			self.win.set_title("Advanced Plotting GUI - " + str(self.args[0]))

		font = {'family' : 'liberation serif',
				'weight' : 'normal',
				'size'   : self.fontSize}

		matplotlib.rc('font', **font)

		if useGTK:
			vbox = gtk.VBox()
			self.win.add(vbox)

		
		# Create plots
		fig = plt.figure()
		fig.title = self.title
		sp = None
#		if self.plot_type == 'scatter':
		sp = scplt()
#		elif self.plot_type == 'time':
#			sp = tplt()
#		else:
#			print "ERROR:"
#			print "plot type: '%s' not supported" % self.plot_type
#			sys.exit(-1)
		#subplots = sp.plot(fig, self.args[0], self.line_style, self.plot_type, self.legendStyle, self.xFormDict, self.sort, self.xnames)
		#subplots = sp.plot(fig, self.args[0], self.line_style, self.plot_type, self.legendStyle, self.xFormDict, self.sort, self.legendLoc)
		subplots = sp.plot(fig, self.args[0], self.line_style, self.plot_type, self.legendStyle, self.xFormDict, self.sort, self.xnames, self.legendLoc, self.markevery)
		plotnum = 0
		for plot in subplots:
			if self.title.has_key(plotnum):
				plot.set_title(self.title[plotnum])

			if self.ylabel.has_key(plotnum):
				plot.set_ylabel(self.ylabel[plotnum])
			else:
				plot.set_ylabel(self.ylabel)
			plot.set_xlabel(self.xlabel)
			#plot.set_markevery(self.markevery)
			self.set_limits(plot, plotnum)
			self.set_ticks(plot)
			plotnum += 1

		if useGTK:
			canvas = FigureCanvas(fig)  # a gtk.DrawingArea
			vbox.pack_start(canvas)

			hbox = gtk.HBox(homogeneous=False, spacing=0)

			toolbar = NavigationToolbar(canvas, self.win)
			toolbox = gtk.VBox()
			toolbox.pack_start(toolbar, True, True)
			hbox.pack_start(toolbox, True, True)

			# add font size text box
			label_fontSize = gtk.Label("Font Size")
			entry_fontSize = gtk.Entry()
			entry_fontSize.set_max_length(4)
			entry_fontSize.set_width_chars(4)
			entry_fontSize.set_text(str(font["size"]))
			entry_fontSize.connect("changed", self.entry_fontSize_changed, [entry_fontSize, canvas])
			hbox.pack_start(label_fontSize, False, False, 3)
			hbox.pack_start(entry_fontSize, False, False, 3)

			cmd_xlog = gtk.Button(label="xlog")
			cmd_xlog.connect("clicked", self.cmd_xlog_handler, [subplots, canvas])
			cmd_ylog = gtk.Button(label="ylog")
			cmd_ylog.connect("clicked", self.cmd_ylog_handler, [subplots, canvas])
			hbox.pack_start(cmd_xlog, False, False, 3)
			hbox.pack_start(cmd_ylog, False, False, 3)

			vbox.pack_start(hbox, False, False, 0)
		
		if self.saveToFile:
			self.doSaveToFile(fig)

		if useGTK:
			self.win.show_all()
			gtk.main()
		else:
			plt.show()

	def set_limits(self, plot, plotnum):
		if hasattr(self, 'ymin') and self.ymin:
			plot.set_ylim(self.ymin, plot.get_ylim()[1])
		if hasattr(self, 'ymax') and self.ymax:
			plot.set_ylim(plot.get_ylim()[0], self.ymax)
		if plotnum == 1 and hasattr(self, 'y1max') and self.y1max:
			plot.set_ylim(plot.get_ylim()[0], self.y1max)
		if hasattr(self, 'xmin') and self.xmin:
			plot.set_xlim(self.xmin, plot.get_xlim()[1])
		if hasattr(self, 'xmax') and self.xmax:
			plot.set_xlim(plot.get_xlim()[0], self.xmax)

#			ymin, ymax = plot.get_ylim()
#			if self
#			ymin = self.ymin
#			ymax = self.ymax
#			plot.set_ylim(0.0,1.0)
#			plot.set_yticks(np.linspace(ymin, ymax, num=11))
		return
	def set_ticks(self, plot):
		if hasattr(self, 'xticks') and self.xticks:
			xmin, xmax = plot.get_xlim()
			plot.set_xticks(np.arange(xmin, xmax+self.xticks, self.xticks))
		if hasattr(self, 'yticks') and self.yticks:
			ymin, ymax = plot.get_ylim()
			plot.set_yticks(np.arange(ymin, ymax+self.yticks, self.yticks))
		return

def main():
	usage = "usage: %prog [options] datafile\n\nThe data format of the datafile is the following (seperator is tab):\nx\ty\tlabel"
	p = OptionParser(usage)
	p.add_option("-t", "--title", dest="title",
		help="changes the title of the plot.")
	p.add_option("", "--title1", dest="title1",
		help="changes the title of the plot1.")
	p.add_option("", "--title2", dest="title2",
		help="changes the title of the plot2.")
	p.add_option("", "--title3", dest="title3",
		help="changes the title of the plot3.")
	p.add_option("", "--title4", dest="title4",
		help="changes the title of the plot4.")

	p.add_option("", "--ymin", dest="ymin",
		help="Limits the y axis on the bottom.")
	p.add_option("", "--y1min", dest="y1min",
		help="Limits the y axis for y1 on the bottom.")
	p.add_option("", "--ymax", dest="ymax",
		help="Limits the y axis on the top.")
	p.add_option("", "--y1max", dest="y1max",
		help="Limits the y axis for y1 on the top.")

	p.add_option("", "--xmin", dest="xmin",
		help="Limits the x axis on the bottom.")
	p.add_option("", "--xmax", dest="xmax",
		help="Limits the x axis on the top.")

	p.add_option("", "--xticks", dest="xticks",
		help="Tick interval for the x axis.")
	p.add_option("", "--yticks", dest="yticks",
		help="Tick interval for the y axis.")
	p.add_option("-n", "--xnames", dest="xnames",
		help="Use this field to specify names of your experiments in bar plotting.")
	p.add_option("-x", "--xlabel", dest="xlabel",
		help="changes the label of the x axes.")
	p.add_option("-y", "--ylabel", dest="ylabel",
		help="changes the label of the y axes.")
	p.add_option("", "--ylabel1", dest="ylabel1",
		help="changes the label of the y1 axes for multi plots.")
	p.add_option("", "--ylabel2", dest="ylabel2",
		help="changes the label of the y2 axes for multi plots.")
	p.add_option("", "--ylabel3", dest="ylabel3",
		help="changes the label of the y3 axes for multi plots.")
	p.add_option("", "--ylabel4", dest="ylabel4",
		help="changes the label of the y4 axes for multi plots.")
	p.add_option("-p", "--plot-type", dest="plot_type",
		help="you can set the plot type to: "+
			"'scatter' (default)."+
			"'time' if the x axis are unix time stamps. "+
			"'date' can be used to show only the date of a timestamp. " +
			"'days' can be used to show days since the beginning of the plot. " +
			"'cdf' for calculating the cumulative distribution function(CDF) over your data. "+
			"'ccdf' for calculating the complementary CDF. "
			"'ecdf' for calculating the empirical CDF. (Not implemented)"
			"'eccdf' for calculating the empirical complementary CDF. "
			"'dict' in combination with the --xdict option to display names for the x-axis coming from a file. "
			)
	p.add_option("-l", "--line-style", dest="line_style",
		help="you can set the line style to '.' for just dots (default) or '-' for connected lines. In case you want to display a scatter plot with many different types of points, try 'multi'. For a better black and white printing use 'bnw' and 'bnw-' for plots with lines and markers.")
	p.add_option("", "--legend", dest="legendStyle", 
		help="Using the 'outside' option plots the legend outside the plotting area. 'below' prints it below the plot")
	p.add_option("", "--fontSize", dest="fontSize", 
		help="Sets the font sizeof the whole plot.")
	p.add_option("", "--xdict", dest="xdict", 
		help="The name of a file containing labels for the x-axis values.")
	p.add_option("", "--sort", dest="sort", 
		help="Default sort option is string, by specifying 'int' one can sort the order of the labels as integers.")
	p.add_option("", "--markevery", dest="markevery", 
		help="Set the markevery property to subsample the plot when using markers. e.g., if markevery=5, every 5-th marker will be plotted.")

	### Adds save to file and engine selection:
	p.add_option("", "--filename", dest="filename", 
		help="If set, the plot is saved to file. See --format to select ratio")
	p.add_option("", "--engine", dest="plotterEngine", 
		help="Select which engine to use for plotting. Currently 'prettyplot' and 'plotterhelper' are available")
	p.add_option("", "--loc", dest="loc", 
		help="Manually force location of legend (for printing). Usese default matplotlib format: "
		"upper right 	1" +
		"upper left 	2" +
		"lower left 	3" +
		"lower right 	4" +
		"right 	5" +
		"center left 	6" +
		"center right 	7" +
		"lower center 	8" +
		"upper center 	9" +
		"center 	10"	
		)
	

	(options, args) = p.parse_args()
	p = Plotter(options, args)
	p.show()

if __name__ == "__main__":
	main()






