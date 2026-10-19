/* a multiline
   comment */
#include <stdio.h>
#include <stdint.h>
#include "local.h"   /* trailing */
#define MAX 10
#define WRAP(x) \
    ((x) + 1)

#if 0
this is skipped
#endif

#ifdef DEBUG
#  define LOG(x) puts(x)
#else
#  define LOG(x)
#endif

static const char *greeting = "hello \"world\"\n";
static char tab = '\t';
static char oct = '\101';
static char hexc = '\x41';

typedef struct point {
    int x;
    unsigned long y;
} point_t;

enum colour { RED, GREEN, BLUE };

double ratio(int a, int b)
{
    double r = 1.5e3;
    float f = .5;
    int h = 0xFFul;
    int o = 0755;
    int d = 42L;
    size_t n = sizeof(point_t);
    return (double)a / (double)b + r + f + h + o + d + n;
}

int main(int argc, char **argv)
{
    point_t p = { .x = 1, .y = 2 };
    register int i;
    volatile int v = 0;

top:
    for (i = 0; i < MAX; i++) {
        switch (i) {
        case 0:
            LOG("zero");
            break;
        case 1: {
            continue;
        }
        default:
            goto top;
        }
    }

    if (argc > 1 && argv != NULL) {
        printf("%s %d\n", greeting, argc);
    } else {
        return 1;
    }
    return 0;
}
