WAIT_FOR_MASTER = while ! kubectl exec -ti mongo-0 -- mongosh --eval "db.runCommand( { isMaster: 1 } ).ismaster" 2>/dev/null | grep -q 'true'; do \
  echo "Waiting for isMaster to be true..."; \
  sleep 1; \
done; \
echo "isMaster is now true."

INIT_MONGO = while ! kubectl exec -ti mongo-0 -- mongosh --eval "rs.initiate().ok" 2>/dev/null | grep -q '1'; do \
  echo "Waiting for rs.initiate() to be true..."; \
  sleep 1; \
done; \
echo "rs.initiate() is now true."

apply_hascheduler:
	kubectl apply -f manifests/hascheduler.yaml

deploy:
	docker build -t hascheduler:0.1.0 .
	kind load docker-image hascheduler:0.1.0 --name kind
	kubectl rollout restart deployment hascheduler

rollout:
	kubectl rollout restart deployment hascheduler

restart_mongo:
	kubectl delete -f manifests/mongo.yaml
	kubectl apply -f manifests/mongo.yaml
	$(INIT_MONGO)
	$(WAIT_FOR_MASTER)

delete_hascheduler:
	kubectl delete -f manifests/hascheduler.yaml

apply_hascheduler:
	kubectl apply -f manifests/hascheduler.yaml

apply_mongo:
	kubectl apply -f manifests/mongo.yaml

get_service_port:
	kubectl get service hascheduler -o jsonpath='{.spec.ports[0].nodePort}'

curl_hascheduler:
	curl $(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane):$(shell kubectl get service hascheduler -o jsonpath='{.spec.ports[0].nodePort}')

get_schedules:
	curl $(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane):$(shell kubectl get service hascheduler -o jsonpath='{.spec.ports[0].nodePort}')/schedules

create_schedules:
	curl -X POST -d '{"name": "test", "type": "duration", "definition": {"interval": "5s"}}' $(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane):$(shell kubectl get service hascheduler -o jsonpath='{.spec.ports[0].nodePort}')/schedules

delete_schedule:
	curl -X DELETE $(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane):$(shell kubectl get service hascheduler -o jsonpath='{.spec.ports[0].nodePort}')/schedules/0eb0e099-1bbf-4d12-be44-58245e0ce4c3

wait_for_mongo:
	$(WAIT_FOR_MASTER)