# A Makefile exercising the lexer's rules.
include config.mk
-include optional.mk

PREFIX ?= /usr/local
CC := gcc
CFLAGS += -Wall -O2
SOURCES = $(wildcard src/*.c)
OBJECTS = $(patsubst src/%.c,build/%.o,$(SOURCES))
VERSION := $(shell git describe --tags 2>/dev/null)
HEADERS = $(filter-out src/internal.h,$(wildcard src/*.h))
SUBDIRS = $(shell (cd src && ls -d */))
RENAMED = $(subst {old},{new},$(HEADERS))
QUOTED = "a string"
SINGLE = 'another string'

$(warning (nested) build check)

export PATH
export BUILD_DIR = build

ifdef DEBUG
CFLAGS += -g
else
CFLAGS += -DNDEBUG
endif

ifeq ($(OS),Windows_NT)
EXT = .exe
endif

.PHONY: all clean install test

all: $(OBJECTS)
	@echo "building $(VERSION)"
	$(CC) $(CFLAGS) -o app$(EXT) $(OBJECTS)

build/%.o: src/%.c $(HEADERS)
	@mkdir -p $(dir $@)
	$(CC) $(CFLAGS) -c $< -o $@

test: all
	./app --self-test && echo "ok"

install: all
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 0755 app $(DESTDIR)$(PREFIX)/bin/

clean:
	rm -rf build app$(EXT)

define GREETING
hello from make
endef
