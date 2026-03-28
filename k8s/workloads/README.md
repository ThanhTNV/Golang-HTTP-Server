# Workloads trong `k8s/workloads`

## Pod
`pod.yaml`: tạo trực tiếp 1 Pod. Nếu Pod chết thì Kubernetes không tự tạo lại (vì không có controller quản lý).

## Deployment
`deployment.yaml`: quản lý replica thông qua Deployment -> ReplicaSet, có self-healing và rolling update.

## ReplicaSet
`replicaset.yaml`: đảm bảo luôn có đúng `N` Pod khớp `selector`.

## DaemonSet
`daemonset.yaml`: chạy 1 Pod trên mỗi Node (thường dùng cho agent như metrics/log/network).

## StatefulSet
`statefulset.yaml`: Pod có định danh ổn định và (tuỳ cấu hình) storage bền qua `volumeClaimTemplates`. File này có thêm headless `Service` để tạo network identity.

## Job
`job.yaml`: chạy công việc đến khi “hoàn thành”, retry theo `backoffLimit`, `restartPolicy: Never`.

## CronJob
`cronjob.yaml`: schedule các `Job` theo cron schedule.
