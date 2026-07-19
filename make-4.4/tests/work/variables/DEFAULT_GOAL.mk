
# Basics.
#
foo: ; @:

ifneq ($(.DEFAULT_GOAL),foo)
$(error )
endif

# Reset to empty.
#
.DEFAULT_GOAL :=

bar: ; @:

ifneq ($(.DEFAULT_GOAL),bar)
$(error )
endif

# Change to a different goal.
#

.DEFAULT_GOAL := baz

baz: ; @echo $@
