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
except:
	print "Could not load gtk"
	exit()

import matplotlib as mpl
import matplotlib.pyplot as plt
import numpy as np
from optparse import OptionParser
from matplotlib.backends.backend_gtkagg import FigureCanvasGTKAgg as FigureCanvas
from matplotlib.backends.backend_gtkagg import NavigationToolbar2GTKAgg as NavigationToolbar


def plot_heatmap(x,y,binsX, binsY, log_scale):
	H, xedges, yedges = np.histogram2d(y, x, bins=[int(binsY), int(binsX)])
	if log_scale:
		H = np.log1p(H)
	return plt.imshow(H, interpolation='hanning', origin='low', extent=[yedges[0], yedges[-1], xedges[0], xedges[-1]], aspect='auto')#, extent=[0,1,0,1])#
	#return plt.imshow(H, interpolation='bilinear', origin='low', extent=[xedges[0], xedges[-1], yedges[0], yedges[-1]], aspect='auto')#, extent=[0,1,0,1])#
	#return plt.imshow(H, interpolation='nearest', origin='low', extent=[xedges[0], xedges[-1], yedges[0], yedges[-1]], aspect='auto')#, extent=[0,1,0,1])#

def plot_heatmap_old2(ax, x,y,binsX, binsY):
	H, xedges, yedges = np.histogram2d(y, x, bins=[int(binsY), int(binsX)])
	im = mpl.image.NonUniformImage(ax, interpolation='bilinear')
	xcenters = xedges[:-1] + 0.5 * (xedges[1:] - xedges[:-1])
	ycenters = yedges[:-1] + 0.5 * (yedges[1:] - yedges[:-1])
	im.set_data(xcenters, ycenters, H)
	ax.images.append(im)
	ax.set_xlim(xedges[0], xedges[-1])
	ax.set_ylim(yedges[0], yedges[-1])
	ax.set_aspect('auto')
	return im

def plot_heatmap_old(ax, x,y,binsX, binsY):
	H, xedges, yedges = np.histogram2d(y, x, bins=[int(binsX), int(binsY)])
	X, Y = np.meshgrid(xedges, yedges)
	im = ax.pcolormesh(X, Y, H)
	return im

def entry_bining_changed_general(data):
	canvas = data[0]
	x = data[1]
	y = data[2]
	cb = data[3]

	entry_biningX = data[4]
	entry_biningY = data[5]
	cmd_log = data[6]

	sizeX = int(entry_biningX.get_value())
	sizeY = int(entry_biningY.get_value())
#	if sizeInt > 2000:
#		print "size too big: " + str(size)
#		return
#	elif sizeInt < 1:
#		print "size too small: " + str(size)
##		return
	im = plot_heatmap(x,y,sizeX, sizeY, cmd_log.get_active())
	cb.update_bruteforce(im)
	canvas.draw()
	return

def entry_biningX_changed(self, widget, data):
	data = self.get_data("data")
	entry_bining_changed_general(data)
	return
def entry_biningY_changed(self, widget, data):
	data = self.get_data("data")
	entry_bining_changed_general(data)
	return
def cmd_log_toggled(self, widget, data):
	data = self.get_data("data")
	entry_bining_changed_general(data)

def main():
	p = OptionParser()
	p.add_option("-t", "--title", dest="title",
		help="changes the title of the plot.")
	p.add_option("-b", "--bins", dest="bins",
		help="The number of bins in a heatmap.")
	p.add_option("", "--xbins", dest="binsX",
		help="The number of bins in a heatmap.")
	p.add_option("", "--ybins", dest="binsY",
		help="The number of bins in a heatmap.")
	p.add_option("-x", "--xlabel", dest="xlabel",
		help="changes the label of the x axes.")
	p.add_option("-y", "--ylabel", dest="ylabel",
		help="changes the label of the y axes.")

	p.add_option("-f", "--filename", dest="filename", 
		help="If set, the plot is saved to file. See --format to select ratio")
	(options, args) = p.parse_args()

	binsX, binsY = (50, 50)
	if hasattr(options, "binsX") and options.binsX:
		binsX = options.binsX
	if hasattr(options, "binsY") and options.binsY:
		binsY = options.binsY
	if hasattr(options, "bins") and options.bins:
		binsX = options.bins
		binsY = options.bins
	title =  "use -t to add a title"
	if hasattr(options, "title") and options.title:
		title = options.title
	xlabel =  "use -x to add a xlabel"
	if hasattr(options, "xlabel") and options.xlabel:
		xlabel = options.xlabel
	ylabel = "use -y to add a ylabel"
	if hasattr(options, "ylabel") and options.ylabel:
		ylabel = options.ylabel


	xl = list()
	yl = list()

	for line in open(options.filename):
		s = line.split('\t')
		xl.append(s[0])
		yl.append(s[1])
	#x = np.random.normal(3, 1, 100000)
	x = np.array(xl,np.float32)
	print "x: " +str(x)
	#y = np.random.normal(1, 1, 100000)
	y = np.array(yl,np.float32)
	print "y: " +str(y)
	#H, xedges, yedges = np.histogram2d(y, x, bins=(xedges, yedges))

	# Create plots
	fig = plt.figure()
	ax = fig.add_subplot(111)
	ax.set_title(title)
	ax.set_xlabel(xlabel)
	ax.set_ylabel(ylabel)
	#H, xedges, yedges = np.histogram2d(y, x, bins=int(options.bins))
	#im = plt.imshow(H, interpolation='hanning', origin='low', extent=[xedges[0], xedges[-1], yedges[0], yedges[-1]])#extent=[0,1,0,1])
	im = plot_heatmap(x, y, binsX, binsY, False)
	cb = fig.colorbar(im)


	win = gtk.Window()
	win.connect("destroy", lambda x: gtk.main_quit())
	win.set_default_size(800,600)
	win.set_title("Advanced Plotting GUI")

	font = {'family' : 'liberation serif',
			'weight' : 'normal',
			'size'   : 12}

	mpl.rc('font', **font)
	vbox = gtk.VBox()
	win.add(vbox)

	canvas = FigureCanvas(fig)  # a gtk.DrawingArea
	vbox.pack_start(canvas)

	hbox = gtk.HBox(homogeneous=False, spacing=0)

	toolbar = NavigationToolbar(canvas, win)
	toolbox = gtk.VBox()
	toolbox.pack_start(toolbar, True, True)
	hbox.pack_start(toolbox, True, True)

	# add font size text box
	label_biningX = gtk.Label("Bin Size X:")
	adjustmentX = gtk.Adjustment(value=float(binsX), lower=1, upper=100000, step_incr=1, page_incr=10, page_size=0)
	entry_biningX = gtk.SpinButton(adjustment=adjustmentX, climb_rate=1, digits=0)
	entry_biningX.set_value(int(binsX))
	entry_biningX.connect("value-changed", entry_biningX_changed, entry_biningX, [])

	label_biningY = gtk.Label("Y:")
	adjustmentY = gtk.Adjustment(value=float(binsY), lower=1, upper=100000, step_incr=1, page_incr=10, page_size=0)
	entry_biningY = gtk.SpinButton(adjustment=adjustmentY, climb_rate=1, digits=0)
	entry_biningY.set_value(int(binsY))
	entry_biningY.connect("value-changed", entry_biningY_changed, entry_biningY, [])
	cmd_log = gtk.CheckButton(label="Log")
	cmd_log.connect("toggled", cmd_log_toggled, cmd_log, [])


	data = [canvas, x, y, cb, entry_biningX, entry_biningY, cmd_log]
	entry_biningX.set_data("data", data)
	entry_biningY.set_data("data", data)
	cmd_log.set_data("data", data)

	hbox.pack_start(label_biningX, False, False, 3)
	hbox.pack_start(entry_biningX, False, False, 5)
	hbox.pack_start(label_biningY, False, False, 3)
	hbox.pack_start(entry_biningY, False, False, 5)
	hbox.pack_start(cmd_log, False, False, 3)

	vbox.pack_start(hbox, False, False, 0)

	win.show_all()
	

if __name__ == "__main__":
	main()
	gtk.main()


