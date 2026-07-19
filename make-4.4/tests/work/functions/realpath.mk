
ifneq ($(realpath .),$(CURDIR))
  $(warning $(realpath .) != $(CURDIR))
endif

ifneq ($(realpath ./),$(CURDIR))
  $(warning $(realpath ./) != $(CURDIR))
endif

ifneq ($(realpath .///),$(CURDIR))
  $(warning $(realpath .///) != $(CURDIR))
endif

.PHONY: all
all: ; @:
