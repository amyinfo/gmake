
all: baz

foo:: f1.h ; @echo foo FIRST
foo:: f2.h ; @echo foo SECOND

bar:: ; @echo aaa; sleep 1; echo aaa done
bar:: ; @echo bbb

baz:: ; @echo aaa
baz:: ; @echo bbb

biz:: ; @echo aaa
biz:: two ; @echo bbb

two: ; @echo two

f1.h f2.h: ; @echo $@

d :: ; @echo ok
d :: d ; @echo oops
