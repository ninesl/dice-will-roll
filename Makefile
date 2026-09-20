WASM_PATH=./build/dicewillroll.wasm
BIN_PATH=./build/dicewillroll
PROFILE_DIR=./build/profiles
CPU_PROFILE_PORT=8080
ALLOC_PROFILE_PORT=8081
GAME_ARGS=
ROCKS_CMD_ARGS=
SHADER_CMD_ARGS=-shader render/shaders/kages/rocks/moon_rock.kage
BENCH_DIR?=/tmp/dice-will-roll-bench
BENCH_PATTERN?=^BenchmarkUpdateRocks
BENCH_TIME?=5s
BENCH_COUNT?=1
BENCH_BIN=${BENCH_DIR}/rocks.test
BENCH_CPU_PROFILE=${BENCH_DIR}/cpu.pprof
BENCH_MEM_PROFILE=${BENCH_DIR}/mem.pprof

.PHONY: help wasm-build bin-build bench profile cpu-flamegraph alloc-flamegraph run-cmds run-rocks-cmd run-shaders-cmd

help:
	@printf 'Available targets:\n'
	@printf '  make bin-build                 Build the native executable\n'
	@printf '  make wasm-build                Build the WebAssembly executable\n'
	@printf '  make bench                     Benchmark and profile packed rock updates\n'
	@printf '  make bench BENCH_TIME=10s      Override time per benchmark\n'
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

bench:
	@set -eu; \
	case '${BENCH_DIR}' in /tmp/*) ;; *) printf 'BENCH_DIR must be below /tmp: %s\n' '${BENCH_DIR}' >&2; exit 1;; esac; \
	rm -rf '${BENCH_DIR}'; \
	mkdir -p '${BENCH_DIR}'; \
	trap 'rm -rf "${BENCH_DIR}"' EXIT INT TERM; \
	printf '\n===== Environment =====\n'; \
	go version; \
	uname -a; \
	printf 'pattern=%s\nbenchtime=%s\ncount=%s\n' '${BENCH_PATTERN}' '${BENCH_TIME}' '${BENCH_COUNT}'; \
	go test -c ./rocks -o '${BENCH_BIN}'; \
	printf '\n===== Benchmarks =====\n'; \
	'${BENCH_BIN}' \
		-test.run '^$$' \
		-test.bench '${BENCH_PATTERN}' \
		-test.benchmem \
		-test.benchtime '${BENCH_TIME}' \
		-test.count '${BENCH_COUNT}' \
		-test.cpuprofile '${BENCH_CPU_PROFILE}' \
		-test.memprofile '${BENCH_MEM_PROFILE}' \
		-test.memprofilerate 1; \
	printf '\n===== CPU Flat Profile =====\n'; \
	go tool pprof -top '${BENCH_BIN}' '${BENCH_CPU_PROFILE}'; \
	printf '\n===== CPU Cumulative Profile =====\n'; \
	go tool pprof -top -cum '${BENCH_BIN}' '${BENCH_CPU_PROFILE}'; \
	printf '\n===== Allocation Profile =====\n'; \
	go tool pprof -sample_index=alloc_space -top '${BENCH_BIN}' '${BENCH_MEM_PROFILE}'; \
	printf '\n===== SIMD Assembly =====\n'; \
	go tool objdump -s 'github.com/ninesl/dice-will-roll/rocks\.(updateRockGroup16|setMouseVelocity16|collideWalls16)' '${BENCH_BIN}'; \
	if command -v perf >/dev/null 2>&1; then \
		printf '\n===== Linux Perf Counters =====\n'; \
		perf stat -d -- '${BENCH_BIN}' \
			-test.run '^$$' -test.bench '${BENCH_PATTERN}' -test.benchmem \
			-test.benchtime '${BENCH_TIME}' -test.count 1 2>&1 || \
			printf 'perf stat failed; check kernel perf permissions.\n'; \
		printf '\n===== Linux Perf Report =====\n'; \
		if perf record -q -g --call-graph dwarf -o '${BENCH_DIR}/perf.data' -- '${BENCH_BIN}' \
			-test.run '^$$' -test.bench '${BENCH_PATTERN}' \
			-test.benchtime '${BENCH_TIME}' -test.count 1; then \
			perf report --stdio -i '${BENCH_DIR}/perf.data'; \
		else \
			printf 'perf record failed; check kernel perf permissions.\n'; \
		fi; \
	else \
		printf '\n===== Linux Perf =====\nperf is not installed; hardware counters skipped.\n'; \
	fi; \
	printf '\nTemporary benchmark files removed from %s\n' '${BENCH_DIR}'

run-rocks-cmd:
	env GOEXPERIMENT=simd go run ./rocks/cmd ${ROCKS_CMD_ARGS}

run-shaders-cmd:
	go run ./render/shaders/cmd ${SHADER_CMD_ARGS}

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
