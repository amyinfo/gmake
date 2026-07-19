
all: ; @echo running rules.

work/features/reinvoke.mk incl.mk: incl-1.mk ; @echo rebuilding $@; echo >> $@

include incl.mk

