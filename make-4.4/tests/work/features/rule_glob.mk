
.PHONY: all a.one a.two a.three
all: a.one* a.t[a-z0-9]o a.th[!q]ee
a.o[Nn][Ee] a.t*: ; @echo $@
