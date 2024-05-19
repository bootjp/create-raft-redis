SHELL := /bin/bash

.PHONY: run prepare clean runA runB runC

# runA runB runC を同時に実行する
run: prepare runA runB runC

prepare:
	@echo "Preparing..."
	rm -rf /tmp/my-raft-cluster/
	mkdir -p /tmp/my-raft-cluster/{nodeA,nodeB,nodeC}
	@echo "Prepared"

runA:
	go run main.go --server_id=nodeA --address=localhost:50051 --redis_address=localhost:63791 --data_dir /tmp/my-raft-cluster/nodeA --initial_peers "nodeB=localhost:50052|localhost:63792,nodeC=localhost:50053|localhost:63793"
runB:
	go run main.go --server_id=nodeB --address=localhost:50052 --redis_address=localhost:63792 --data_dir /tmp/my-raft-cluster/nodeB --initial_peers "nodeA=localhost:50051|localhost:63791,nodeC=localhost:50053|localhost:63793"
runC:
	go run main.go --server_id=nodeC --address=localhost:50053 --redis_address=localhost:63793 --data_dir /tmp/my-raft-cluster/nodeC --initial_peers "nodeA=localhost:50051|localhost:63791,nodeB=localhost:50052|localhost:63792"
