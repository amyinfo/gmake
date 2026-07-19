
.SUFFIXES:

.PHONY: all
all: foo

foo: bar.x
	@echo cp $< $@
	@echo "" > $@
