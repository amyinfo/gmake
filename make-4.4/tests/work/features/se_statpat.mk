
.SECONDEXPANSION:
.DEFAULT: ; @echo '$@'

foo.a foo.b: foo.%: bar.% baz.%
foo.a foo.b: foo.%: biz.% | buz.%

foo.a foo.b: foo.%: $$@.1 \
                    $$<.2 \
                    $$(addsuffix .3,$$^) \
                    $$(addsuffix .4,$$+) \
                    $$|.5 \
                    $$*.6
