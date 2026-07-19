
all: one two
	$(warning in $@ line 3)
	@true
	$(warning in $@ line 5)

one two:
	$(warning in $@ line 8)
	@true
	$(warning in $@ line 10)
