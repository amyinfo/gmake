final-a: interm-a ; echo >> $@
final-b: interm-b ; echo >> $@
interm-a:: orig1-a ; echo >> $@
interm-a:: orig2-a ; echo >> $@
interm-b:: orig1-b ; echo >> $@
interm-b:: orig2-b ; echo >> $@
