default:
	go run .

test-straight:
	go test -v -run TestFindBestSingleConsecutive

test:
	go test -v
