
deploy:
	docker build -t hascheduler:0.1.0 .
	kind load docker-image hascheduler:0.1.0 --name kind
	kubectl rollout restart deployment hascheduler

delete_hascheduler:
	kubectl delete -f manifests/hascheduler.yaml

apply_hascheduler:
	kubectl apply -f manifests/hascheduler.yaml

apply_mongo:
	kubectl apply -f manifests/mongo.yaml