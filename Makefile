.PHONY: build load-image deploy rollout mongo-rs-init mongo-restart mongo-apply mongo-delete hascheduler-delete hascheduler-apply get-schedules create-schedule update-schedule delete-schedule shutdown kind-context kind-up up

# Variables
IMAGE_NAME = hascheduler:0.1.0
KIND_CLUSTER_NAME = kind
KIND_CONTROL_PLANE_IP = $(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane)
HASCHEDULER_PORT = $(shell kubectl get service hascheduler -o jsonpath='{.spec.ports[0].nodePort}')
HASCHEDULER_URL = $(KIND_CONTROL_PLANE_IP):$(HASCHEDULER_PORT)

# MongoDB initialization commands
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

# Build Docker image
build:
	docker build -t $(IMAGE_NAME) .

# Load Docker image into kind cluster
load-image:
	kind load docker-image $(IMAGE_NAME) --name $(KIND_CLUSTER_NAME)

# Deploy application
deploy: build load-image
	kubectl rollout restart deployment hascheduler

# Rollout restart
rollout:
	kubectl rollout restart deployment hascheduler

# Initialize MongoDB replica set
mongo-rs-init:
	$(INIT_MONGO)
	$(WAIT_FOR_MASTER)

# Restart MongoDB
mongo-restart: mongo-delete mongo-apply mongo-rs-init

# Apply MongoDB manifests
mongo-apply:
	kubectl apply -f manifests/mongo.yaml

# Delete MongoDB manifests
mongo-delete:
	kubectl delete -f manifests/mongo.yaml

# Delete hascheduler manifests
hascheduler-delete:
	kubectl delete -f manifests/hascheduler.yaml

# Apply hascheduler manifests
hascheduler-apply:
	kubectl apply -f manifests/hascheduler.yaml

# Get schedules
get-schedules:
	curl $(HASCHEDULER_URL)/schedules | jq

# Create a schedule
create-schedule:
	curl -X POST -d @payloads/create_schedule.json $(HASCHEDULER_URL)/schedules | jq

# Update a schedule
update-schedule:
	curl -X PUT -d @payloads/update_schedule.json $(HASCHEDULER_URL)/schedules/$(ID) | jq

# Delete a schedule
delete-schedule:
	curl -X DELETE $(HASCHEDULER_URL)/schedules/$(ID) | jq

# Shutdown kind cluster
shutdown:
	kind delete cluster

# Set kind context
kind-context:
	kubectl config use-context kind-$(KIND_CLUSTER_NAME)
	kubectl cluster-info --context kind-$(KIND_CLUSTER_NAME)

# Create kind cluster
kind-up:
	kind create cluster

# Start the demo
up: kind-up kind-context mongo-apply mongo-rs-init build load-image hascheduler-apply