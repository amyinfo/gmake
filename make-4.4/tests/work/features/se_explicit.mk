
ifdef SE
  .SECONDEXPANSION:
endif
foo$$bar: bar$$baz bar$$biz ; @echo '$@ : $^'
PRE = one two
bar$$baz: $$(PRE)
baraz: $$(PRE)
PRE = three four
.DEFAULT: ; @echo '$@'
