WASM_PATH=./build/dicewillroll.wasm
BIN_PATH=./build/dicewillroll
PROFILE_DIR=./build/profiles
CPU_PROFILE_PORT=8080
ALLOC_PROFILE_PORT=8081
GAME_ARGS=
ROCKS_CMD_ARGS=
SHADER_CMD_ARGS=-shader cmd/background.kage

.PHONY: help wasm-build bin-build profile cpu-flamegraph alloc-flamegraph run-cmds run-rocks-cmd run-shaders-cmd

help:
	@printf 'Available targets:\n'
	@printf '  make bin-build                 Build the native executable\n'
	@printf '  make wasm-build                Build the WebAssembly executable\n'
	@printf '  make profile                   Run the game, capture profiles, and serve both pprof UIs\n'
	@printf '  make profile GAME_ARGS="-rocks 5000"\n'
	@printf '                                 Pass game flags to a profiling run\n'
	@printf '  make cpu-flamegraph            Serve CPU flame graph at http://localhost:${CPU_PROFILE_PORT}/ui/flamegraph\n'
	@printf '  make alloc-flamegraph          Serve allocations at http://localhost:${ALLOC_PROFILE_PORT}/ui/flamegraph\n'
	@printf '  make run-rocks-cmd             Run the packed rock sprite viewer\n'
	@printf '  make run-shaders-cmd           Run the shader viewer\n'
	@printf '  make run-cmds                  Run both viewers sequentially\n'
	@printf '\nAfter closing a profiling run, open the two Flame Graph URLs printed by make.\n'

wasm-build:
	mkdir -p $(dir ${WASM_PATH})
	env GOOS=js GOARCH=wasm go build -o ${WASM_PATH} .

bin-build:
	go build -o ${BIN_PATH} .

run-rocks-cmd:
	env GOEXPERIMENT=simd go run ./rocks/cmd ${ROCKS_CMD_ARGS}

run-shaders-cmd:
	cd render/shaders/cmd && go run . ${SHADER_CMD_ARGS}

run-cmds: run-rocks-cmd run-shaders-cmd

profile:
	mkdir -p ${PROFILE_DIR}
	go run . -cpuprofile=${PROFILE_DIR}/cpu.pprof -memprofile=${PROFILE_DIR}/mem.pprof ${GAME_ARGS}
	@printf '\nGame closed. Open the CPU flame graph:\n  http://localhost:${CPU_PROFILE_PORT}/ui/flamegraph\n'
	@printf 'Open the allocation flame graph:\n  http://localhost:${ALLOC_PROFILE_PORT}/ui/flamegraph\n'
	@printf 'Press Ctrl-C when you are finished.\n\n'
	@trap 'kill $$cpu_pid $$alloc_pid 2>/dev/null || true' EXIT INT TERM; \
		go tool pprof -no_browser -http=:${CPU_PROFILE_PORT} ${PROFILE_DIR}/cpu.pprof & cpu_pid=$$!; \
		go tool pprof -no_browser -http=:${ALLOC_PROFILE_PORT} -sample_index=alloc_space ${PROFILE_DIR}/mem.pprof & alloc_pid=$$!; \
		wait

cpu-flamegraph:
	@printf 'Open http://localhost:${CPU_PROFILE_PORT}/ui/flamegraph\n'
	go tool pprof -no_browser -http=:${CPU_PROFILE_PORT} ${PROFILE_DIR}/cpu.pprof

alloc-flamegraph:
	@printf 'Open http://localhost:${ALLOC_PROFILE_PORT}/ui/flamegraph\n'
	go tool pprof -no_browser -http=:${ALLOC_PROFILE_PORT} -sample_index=alloc_space ${PROFILE_DIR}/mem.pprof
