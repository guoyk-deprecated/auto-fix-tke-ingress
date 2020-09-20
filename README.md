# auto-fix-tke-ingress

automatically fix tke ingress

## Usage

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: auto-fix-tke-ingress
  namespace: autoops
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: auto-fix-tke-ingress
rules:
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["watch"]
  - apiGroups: [""]
    resources: ["ingresses"]
    verbs: ["watch", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: auto-fix-tke-ingress
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: auto-fix-tke-ingress
subjects:
  - kind: ServiceAccount
    name: auto-fix-tke-ingress
    namespace: autoops
---
apiVersion: v1
kind: Service
metadata:
  name: auto-fix-tke-ingress
  namespace: autoops
spec:
  ports:
    - port: 42
      name: life
  clusterIP: None
  selector:
    app: auto-fix-tke-ingress
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: auto-fix-tke-ingress
  namespace: autoops
spec:
  selector:
    matchLabels:
      app: auto-fix-tke-ingress
  serviceName: auto-fix-tke-ingress
  replicas: 1
  template:
    metadata:
      labels:
        app: auto-fix-tke-ingress
    spec:
      serviceAccount: auto-fix-tke-ingress
      containers:
        - name: auto-fix-tke-ingress
          image: guoyk/auto-fix-tke-ingress
          imagePullPolicy: Always
```

Add annotation to Ingress

```yaml
net.guoyk.auto-fix-tke-ingress/enabled: "true"
```

## Credits

Guo Y.K., MIT License
