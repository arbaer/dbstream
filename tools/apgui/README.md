APGUI  - Advanced Plotting Graphical User Interface

The tool agpui is used to plot data in multiple formats, among them are line plots, scatter plots and bar plots.

EXAMPLES:

Line plots:
	./apgui.py -l - -x "Exec time" -y "Cache usage" -t "Line plot" lines.txt

	It is also possible to have more complex plots, with e.g. more colors of the lines and the legend outside the plot

	./apgui.py -l multismall- --legend outside -x "Exec time" -y "Cache usage" -t "Line plot with different colors and legend outside" lines.txt	

	In case your X axis contains UNIX timestamps you can use the -p time option.


Multi line plots:
	You can also plot several plots in one figure with a synchronized X axis

	./apgui.py -l multismall- --legend single -x "Exec time" -y "Cache usage" -t "Multi line plots with synchronized X axis" lines_multi.txt	



Fill plot:
	./apgui.py -l - -x "Exec time" -y "Cache usage" -t "Fill plot" -p fill lines.txt


Stack plot:
	./apgui.py -l - -x "Exec time" -y "Cache usage" -t "Stack plot" -p stack lines.txt


Bar plot:
	Please note that the bar plots can be use with or without confidence intervals.

	./apgui.py -p bar bar.txt


Heatmaps:
	For the simples form of heatmaps you can use this:

	./heatmap.py -f heat_data.txt

	You can also specify the number of bins (also separately for X and Y axis see --help):
	./heatmap.py -f heat_data.txt --bins 50

	Please note that at the moment the interpolation method is set to "hanning", if you prefer no interpolation change it to "nearest".


AUTHORS:

Arian Bär (main contributer) arian.baer@gmail.com 
Thomas Paulin