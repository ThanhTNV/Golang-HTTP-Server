# Resources trong `k8s/resources`

Thư mục này chứa các ví dụ “resource” Kubernetes thường dùng (không phải workload controller).

## Cách dùng nhanh

```sh
kubectl apply -f k8s/resources/configmap.yaml
kubectl delete -f k8s/resources/configmap.yaml
```

Hoặc apply “1 phát” tất cả resource (single manifest):

```sh
kubectl apply -f k8s/resources/all.yaml
kubectl delete -f k8s/resources/all.yaml
```

## Service
`service-clusterip.yaml`: tạo `Service` loại `ClusterIP` để expose Pod trong nội bộ cluster.

Điểm chính:
- Dùng `selector` để route traffic tới các Pod có label phù hợp.
- Service tạo ra stable virtual IP + DNS name trong cluster.

Các biến thể khác:
- `service-nodeport.yaml`: expose ra cổng trên mỗi Node (demo cố định `nodePort: 32080`).
- `service-loadbalancer.yaml`: nhờ cloud provider/LB controller cấp external IP (thường dùng trên managed K8s).

## ConfigMap
`configmap.yaml`: cấu hình dạng key/value.

Ví dụ dùng:
- `pod-with-configmap-volume.yaml`: vừa inject env bằng `envFrom.configMapRef`, vừa mount `ConfigMap` thành volume tại `/etc/app-config`.

## Secret
`secret.yaml`: giống ConfigMap về “cách gắn vào Pod”, nhưng dùng cho dữ liệu nhạy cảm (password/token/key...).

Ví dụ dùng:
- `pod-with-secret.yaml`: vừa inject env bằng `envFrom.secretRef`, vừa mount `Secret` thành volume tại `/etc/app-secret`.

## PersistentVolume (PV) + PersistentVolumeClaim (PVC)
`pv-pvc-hostpath.yaml` gồm 2 resource:
- `PersistentVolume`: volume vật lý/logic mà cluster có thể dùng.
- `PersistentVolumeClaim`: “yêu cầu” storage từ ứng dụng; scheduler/binder sẽ bind PVC với PV phù hợp.

Lưu ý:
- Mẫu này dùng `hostPath` để demo đơn giản; phù hợp kiểu local dev (minikube/kind) hơn là production.
- `storageClassName: manual` để PV/PVC match nhau dễ đọc.

Ví dụ dùng:
- `pod-with-pvc.yaml`: mount PVC `k8s-pod-pvc` vào `/data`.

## emptyDir (volume thuần túy trong Pod)
`pod-with-emptydir.yaml`: volume dạng `emptyDir` chỉ tồn tại trong vòng đời Pod.

Điểm chính:
- Khi Pod chạy, dữ liệu trong `/data` được dùng bình thường.
- Khi Pod bị xóa/được tạo lại, `emptyDir` sẽ mất dữ liệu.
