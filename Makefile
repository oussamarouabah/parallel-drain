build:
	go build -o kubectl-parallel_drain main.go
	sudo mv kubectl-parallel_drain /usr/local/bin/
