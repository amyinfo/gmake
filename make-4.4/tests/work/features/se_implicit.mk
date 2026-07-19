
.SECONDEXPANSION:
.DEFAULT: ; @echo '$@'

foo.a: bar baz

foo.a: biz | buz

foo.%: 1.$$@ \
       2.$$< \
       $$(addprefix 3.,$$^) \
       $$(addprefix 4.,$$+) \
       5.$$| \
       6.$$* ; @:

1.foo.a \
2.bar \
3.bar \
3.baz \
3.biz \
4.bar \
4.baz \
4.biz \
5.buz \
6.a: ; @echo '$@'

