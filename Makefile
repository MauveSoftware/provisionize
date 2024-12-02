all: server

.PHONY: server
server:
	docker build --rm --no-cache --tag eu.gcr.io/mauve-cloud/provisionize .
	docker push eu.gcr.io/mauve-cloud/provisionize
