
foo:; @echo Executing rule FOO

.DEFAULT: ; @$(MAKE) -f defsub.mk $@
