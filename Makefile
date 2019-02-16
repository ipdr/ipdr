.PHONY: test
test:
	go test -v ./... && echo 'ALL PASS'


.PHONY: test/install-deps
test/install-deps:
	set -x
	# configure hosts
	echo '127.0.0.1       docker.localhost' >> /etc/hosts
	# install Docker
	VER="18.03.1-ce"
	curl -L -o /tmp/docker-$VER.tgz https://download.docker.com/linux/static/stable/x86_64/docker-$VER.tgz
	tar -xz -C /tmp -f /tmp/docker-$VER.tgz
	sudo mv /tmp/docker/* /usr/bin
	# install IPFS
	wget https://dist.ipfs.io/go-ipfs/v0.4.14/go-ipfs_v0.4.14_linux-amd64.tar.gz -O /tmp/go-ipfs.tar.gz
	cd /tmp
	tar xvfz go-ipfs.tar.gz
	sudo cp go-ipfs/ipfs /usr/bin/
	ipfs version
	# run IPFS daemon
	ipfs init
	ipfs config Addresses.API /ip4/0.0.0.0/tcp/5001
	ipfs config Addresses.Gateway /ip4/0.0.0.0/tcp/9001
	ipfs daemon &

.PHONY: clean
clean:
	rm -f docker/*.tar
	rm ipfs/tmp_data

