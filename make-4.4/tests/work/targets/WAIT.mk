
all: .WAIT pre1 .WAIT pre2 | .WAIT pre3 ; @echo '<=$< ^=$^ ?=$? +=$+ |=$|'
pre1 pre2 pre3:;

# This is just here so we don't fail with older versions of make
.WAIT:
