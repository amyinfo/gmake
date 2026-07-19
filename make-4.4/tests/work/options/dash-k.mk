all: ; @echo hi
include ifile
ifile: no-such-file; exit 1
