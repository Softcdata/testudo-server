#!/bin/bash
set -e

# Destination registry namespace, for example:
#   REGISTRY=registry.example.com/my-namespace ./push_velero_huawei.sh
REGISTRY="${REGISTRY:-registry.example.com/disaster}"

echo "Pushing Velero images to $REGISTRY..."

# Helper function using Skopeo
copy_to_huawei() {
    local src=$1
    local dest_name=$2
    local dest="$REGISTRY/$dest_name"
    
    echo "Copying $src to $dest..."
    # --all copies the manifest list and all architectures (amd64, arm64, etc.)
    # --src-tls-verify=false (optional if src has issues, but docker.1ms.run usually fine)
    # --dest-tls-verify=true (Huawei Cloud uses valid certs)
    skopeo copy --all \
        --src-tls-verify=false \
        docker://"$src" \
        docker://"$dest"
}

# Velero Core
copy_to_huawei "docker.1ms.run/velero/velero:v1.17.0" "velero:v1.17.0"

# Velero AWS Plugin
copy_to_huawei "docker.1ms.run/velero/velero-plugin-for-aws:v1.13.0" "velero-plugin-for-aws:v1.13.0"

echo "Done pushing Velero images!"
