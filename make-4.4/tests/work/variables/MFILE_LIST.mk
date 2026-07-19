
m1 := $(MAKEFILE_LIST)
include incl2
m3 := $(MAKEFILE_LIST)

all:
	@echo $(m1)
	@echo $(m2)
	@echo $(m3)
