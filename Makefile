LOGS_DIR=./_logs

all: help

help:
	@echo "build          - build binary"
	@echo "N={num} run    - run N instances of hashgossiper. Save output into files in _logs dir"
	@echo "watcher        - send monitoring command over multicast and wait for answers"
	@echo "kill           - send kill command over multicast"
	@echo "clean          - send kill and rm logs"

build:
	go build hashgossip.go

run:
	for i in {1..${N}}; do ./hashgossip 1>$(LOGS_DIR)/$$i.log 2>&1 & done

kill:
	./hashgossip -killer

clean: kill
	rm -f $(LOGS_DIR)/*.log

watcher:
	./hashgossip -watcher 2>&1
