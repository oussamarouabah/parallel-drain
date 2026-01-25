build:
	go build -o kubectl-parallel_drain main.go
	sudo cp kubectl-parallel_drain /usr/local/bin/
