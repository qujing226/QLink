.PHONY: gateway cluster stop status logs

gateway:
	@bash scripts/start_gateway.sh

cluster:
	@bash scripts/start_cluster.sh

stop:
	@bash scripts/stop_all.sh

status:
	@echo "Node status:" && curl -sS http://localhost:8080/api/v1/node/status || true && echo && \
	 echo "Cluster status:" && curl -sS http://localhost:8080/api/v1/cluster/status || true

logs:
	@ls -al logs || true