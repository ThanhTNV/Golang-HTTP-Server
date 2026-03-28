kubectl apply -f workloads/statefulset.yaml    # includes postgres-svc
kubectl apply -f workloads/deployment.yaml
kubectl apply -f workloads/daemonset.yaml
kubectl apply -f workloads/replicaset.yaml
kubectl apply -f workloads/job.yaml
kubectl apply -f workloads/cronjob.yaml