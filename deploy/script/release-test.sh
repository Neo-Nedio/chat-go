#!/bin/bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
project_root="$(cd -- "${script_dir}/../.." && pwd)"

cd "${project_root}"

# 先启动 etcd、Redis、MySQL，确保 RPC 容器加入同一个 chat 网络。
docker compose up -d etcd redis mysql

need_start_server_shell=(
  # rpc
  user-rpc-test.sh

  # api
)

for i in "${need_start_server_shell[@]}" ; do
    chmod +x "${script_dir}/${i}"
    "${script_dir}/${i}"
done


docker ps

docker exec -it etcd etcdctl get --prefix ""
