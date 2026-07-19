#include <string.h>
#include <stdio.h>

#include "gnumake.h"

char *getenv (const char*);

int plugin_is_GPL_compatible;

int testapi_gmk_setup ();

static char *
test_eval (const char *buf)
{
    gmk_eval (buf, 0);
    return NULL;
}

static char *
test_expand (const char *val)
{
    return gmk_expand (val);
}

static char *
test_noexpand (const char *val)
{
    char *str = gmk_alloc (strlen (val) + 1);
    strcpy (str, val);
    return str;
}

static char *
func_test (const char *funcname, unsigned int argc, char **argv)
{
    char *mem;

    if (strcmp (funcname, "test-expand") == 0)
        return test_expand (argv[0]);

    if (strcmp (funcname, "test-eval") == 0)
        return test_eval (argv[0]);

    if (strcmp (funcname, "test-noexpand") == 0)
        return test_noexpand (argv[0]);

    mem = gmk_alloc (sizeof ("unknown"));
    strcpy (mem, "unknown");
    return mem;
}

int
testapi_gmk_setup (const gmk_floc *floc)
{
    const char *verbose = getenv ("TESTAPI_VERBOSE");

    gmk_add_function ("test-expand", func_test, 1, 1, GMK_FUNC_DEFAULT);
    gmk_add_function ("test-noexpand", func_test, 1, 1, GMK_FUNC_NOEXPAND);
    gmk_add_function ("test-eval", func_test, 1, 1, GMK_FUNC_DEFAULT);
    gmk_add_function ("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.", func_test, 0, 0, 0);

    if (verbose)
      {
        printf ("testapi_gmk_setup\n");

        if (verbose[0] == '2')
          printf ("%s:%lu\n", floc->filenm, floc->lineno);
      }

    if (getenv ("TESTAPI_KEEP"))
      return -1;

    return 1;
}
