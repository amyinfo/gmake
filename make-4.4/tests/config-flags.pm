# This is a -*-perl-*- script
# Minimal config for gmake test suite

%CONFIG_FLAGS = (
    AM_LDFLAGS      => '',
    AR              => 'ar',
    CC              => 'cc',
    CFLAGS          => '',
    CPP             => 'cc -E',
    CPPFLAGS        => '',
    GUILE_CFLAGS    => '',
    GUILE_LIBS      => '',
    LDFLAGS         => '',
    LIBS            => '',
    USE_SYSTEM_GLOB => ''
);

1;
