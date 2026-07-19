
.LIBPATTERNS = mtest_%.a
all: -lfoo ; @echo "build $@ from $<"
