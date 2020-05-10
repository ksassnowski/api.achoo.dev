dist: export GOOS=linux
dist: export GOARCH=amd64
dist: clean
	go build -o dist/pollen-api

test:
	go test ./...

clean:
	rm -rf dist/*