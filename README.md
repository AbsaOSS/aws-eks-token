**aws-get-token**

Is a golang implementation of `aws eks get-token` functionality, which is part of aws cli v2 (python implementation). The aim of this project is to simplify maintenance and dependency management while using kubectl within contenerized environment. Such as terraform kubectl provider running in tf-runner kubernetes pod.

Following flags are supported:

```code
--region: override AWS region
--cluster-name: EKS cluster name or ID to retrieve a token for (required)
--role-arn: AWS role to assume (in arn format)
```

