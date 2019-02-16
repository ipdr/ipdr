.PHONY: test
test:
	go test -v ./... && echo 'ALL PASS'

.PHONY: clean
clean:
	rm -f docker/*.tar
	rm ipfs/tmp_data

