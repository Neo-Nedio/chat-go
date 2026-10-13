#!/bin/bash
set -euo pipefail

reso_addr='registry.cn-hangzhou.aliyuncs.com/sai-im/user-rpc-dev'
tag='latest'

container_name="sai-im-user-rpc-test"

docker rm -f "${container_name}" 2>/dev/null || true
docker pull ${reso_addr}:${tag}

# compose 使用固定的 chat 网络，容器内依赖服务通过 etcd/mysql/redis 访问。
docker run -p 10000:8080 --network chat --restart unless-stopped \
  --name="${container_name}" -d "${reso_addr}:${tag}"
