
$(info infile)
BAR = bar
all: ; @echo all
recurse: ; @$(MAKE) -f work/options/eval.mk && echo recurse
