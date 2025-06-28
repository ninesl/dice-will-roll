build-all:
	mkdir -p build
	GOOS=linux GOARCH=amd64 go build -o build/dice-will-roll-linux-amd64
	GOOS=windows GOARCH=amd64 go build -o build/dice-will-roll-windows-amd64.exe
	zip -j -9 build/dice-will-roll-windows.zip build/dice-will-roll-windows-amd64.exe
#	GOOS=darwin GOARCH=amd64 go build -o build/dice-will-roll-darwin-amd64
#	GOOS=darwin GOARCH=arm64 go build -o build/dice-will-roll-darwin-arm64

clean:
	rm -rf build

run:
	go run main.go
