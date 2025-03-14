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

build:
	docker build -t hascheduler:0.1.0 .

load_image:
	kind load docker-image hascheduler:0.1.0 --name kind

deploy: build load_image
	kubectl rollout restart deployment hascheduler

rollout:
	kubectl rollout restart deployment hascheduler

mongo_rs_init:
	$(INIT_MONGO)
	$(WAIT_FOR_MASTER)

mongo_restart: mongo_delete mongo_apply mongo_rs_init

mongo_apply:
	kubectl apply -f manifests/mongo.yaml

mongo_delete:
	kubectl delete -f manifests/mongo.yaml

hascheduler_delete:
	kubectl delete -f manifests/hascheduler.yaml

hascheduler_apply:
	kubectl apply -f manifests/hascheduler.yaml

get_schedules:
	curl $(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane):$(shell kubectl get service hascheduler -o jsonpath='{.spec.ports[0].nodePort}')/schedules

create_schedule:
	curl -X POST -d '{"name": "test", "type": "duration", "definition": {"interval": "5s"}}' $(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane):$(shell kubectl get service hascheduler -o jsonpath='{.spec.ports[0].nodePort}')/schedules

update_schedule:
	curl -X PUT -d '{"name": "test", "type": "duration", "definition": {"interval": "3s"}}' $(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane):$(shell kubectl get service hascheduler -o jsonpath='{.spec.ports[0].nodePort}')/schedules/$(ID)

delete_schedule:
	curl -X DELETE $(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane):$(shell kubectl get service hascheduler -o jsonpath='{.spec.ports[0].nodePort}')/schedules/$(ID)

shutdown:
	kind delete cluster

kind_context:
	kubectl config use-context kind-kind
	kubectl cluster-info --context kind-kind

kind_up:
	kind create cluster

up: kind_up kind_context mongo_apply mongo_rs_init build load_image hascheduler_apply