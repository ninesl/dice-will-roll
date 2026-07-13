WASM_PATH=./build/dicewillroll.wasm
BIN_PATH=./build/dicewillroll

wasm-build:
	mkdir -p $(dir ${WASM_PATH})
	env GOOS=js GOARCH=wasm go build -o ${WASM_PATH} .

bin-build:
	go build -o ${BIN_PATH} .
