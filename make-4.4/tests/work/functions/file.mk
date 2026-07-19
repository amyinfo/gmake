
define A
a
b
endef
B = c d
$(file >file.out,$(A))
$(foreach L,$(B),$(file >>     file.out,$L))
x:;@echo hi; cat file.out
