TARGET = multiping

.PHONY: all
all: $(TARGET)

$(TARGET):
	go build -o $@
	sudo setcap cap_net_raw+ep $<

.PHONY: clean
clean:
	rm -f $(TARGET) $(TARGET).log

.PHONY: test
test: all
	./$(TARGET) golang.org cloudflare.com 2>$(TARGET).log
