
.POSIX:

demo1!=printf '  1   2 3\n4\n\n5 \n \n 6\n\n\n\n'
demo2 != printf '7 8\n '
demo3 != printf '$$(demo2)'
demo4 != printf ' 2 3 \n'
demo5 != printf ' 2 3 \n\n'
all: ; @echo "<$(demo1)> <$(demo2)> <$(demo3)> <$(demo4)> <${demo5}>"
