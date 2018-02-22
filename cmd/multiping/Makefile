TARGET = multiping

include ../common.mk

.PHONY: test
test: all
	./$(TARGET) golang.org cloudflare.com 2>$(TARGET).log
