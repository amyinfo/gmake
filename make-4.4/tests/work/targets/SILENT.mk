
.PHONY: M a b
M: a b
.SILENT : b
a b: ; echo $@
