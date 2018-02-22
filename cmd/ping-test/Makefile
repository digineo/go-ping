TARGET = ping-test

include ../common.mk

.PHONY: test
test: all
	@truncate -s0 $(TARGET).log
	@./$(TARGET) golang.org      2>>$(TARGET).log
	@./$(TARGET) cloudflare.com  2>>$(TARGET).log

	@echo error log:
	@cat $(TARGET).log
