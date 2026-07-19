define Y
  all:: ; @echo $AA
  A = B
endef

X = $(eval $(value Y))

$(eval $(shell echo A = A))
$(eval $(Y))
$(eval A = C)
$(eval $(X))
