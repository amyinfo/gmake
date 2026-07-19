
foo := bletch garf
auto_var = undefined CC MAKETEST MAKE foo CFLAGS WHITE @
av = $(foreach var, $(auto_var), $(origin $(var)) )
override WHITE := BLACK
.RECIPEPREFIX = >
all: auto
> @echo $(origin undefined)
> @echo $(origin CC)
> @echo $(origin MAKETEST)
> @echo $(origin MAKE)
> @echo $(origin foo)
> @echo $(origin CFLAGS)
> @echo $(origin WHITE)
> @echo $(origin @)
auto :
> @echo $(av)
