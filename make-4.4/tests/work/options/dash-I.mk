
main_makefile := $(firstword $(MAKEFILE_LIST))
all:
	@echo There should be no errors for this makefile
include ifile.mk
