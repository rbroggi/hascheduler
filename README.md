# hascheduler

Highly available and distributed scheduler demo using [leader-election](https://github.com/rbroggi/leaderelection) and 
[MongoDB change streams](https://www.mongodb.com/docs/manual/changeStreams/).

## Create a kind Cluster (if you haven't already):

```bash
kind create cluster
```

Apply the Updated Manifests:

```bash
kubectl apply -f manifests/mongo.yaml
kubectl apply -f manifests/hascheduler.yaml
kubectl apply -f manifests/prometheus.yaml
kubectl apply -f manifests/grafana.yaml
```

Verify Deployments:

```bash
kubectl get deployments
kubectl get pods
kubectl get services
```
Ensure all deployments and pods are running.

Access Prometheus:
Get the NodePort for the Prometheus service:

```bash
kubectl get service prometheus
```

Access Prometheus in your browser using http://localhost:<prometheus-nodeport>.

Access Grafana:
Get the NodePort for the Grafana service:

```bash
kubectl get service grafana
```
Access Grafana in your browser using http://localhost:<grafana-nodeport>.
Login with user admin and password admin.

Add Prometheus as a Data Source in Grafana:

In Grafana, go to "Configuration" (gear icon) -> "Data Sources".
Click "Add data source".
Select "Prometheus".
In the URL field, enter http://prometheus:9090.
Click "Save & test". You should see "Data source is working".
Now you can create dashboards to visualize your Go application metrics.
Update Your Go Application:

You'll need to modify your Go application to connect to MongoDB instead of Redis.
Use a MongoDB Go driver (e.g., go.mongodb.org/mongo-driver/mongo).
Use the MONGO_URI environment variable to get the connection string.
Ensure your health endpoint now checks the connection to mongo.
Ensure to expose metrics, if needed.

```bash
docker build -t hascheduler:latest .
```

Load image into kind

```bash
kind load docker-image hascheduler:latest
```