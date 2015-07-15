all:
	sed -i "s/VERSION/${VERSION}/g" master-multi.json master.json
	docker build -t wizardcxy/hyperkube:${VERSION} .

.PHONY: all
