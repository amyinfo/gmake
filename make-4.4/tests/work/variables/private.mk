
a:
F = g
a: F = a
b: private F = b

a b c: ; @echo $@: F=$(F)
a: b
b: c
