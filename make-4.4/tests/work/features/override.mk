
X = start
override recur = $(X)
override simple := $(X)
X = end
all: ; @echo "$(recur) $(simple)"
