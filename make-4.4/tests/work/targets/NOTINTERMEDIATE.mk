
hello.z:
%.z: %.x; touch $@
%.x: ;
.NOTINTERMEDIATE: %.q %.x
