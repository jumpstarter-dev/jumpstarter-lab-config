#!/bin/sh

NAMESPACE=${NAMESPACE:-jumpstarter-lab}
SA=${SA:-jumpstarter-lab-config}
CONTEXT=${CONTEXT:-jumpstarter-lab}

# explain to stderr

echo "Writing kubeconfig for ${CONTEXT} in ${NAMESPACE} with ${SA} to stdout" >&2
echo "Please tweak the NAMESPACE, SA, and CONTEXT env variables if needed" >&2
echo "NAMESPACE: ${NAMESPACE}" >&2
echo "SA: ${SA}" >&2
echo "CONTEXT: ${CONTEXT}" >&2

SECRET_NAME="${SA}-token"
TOKEN=$(kubectl get secret $SECRET_NAME -n ${NAMESPACE} -o jsonpath='{.data.token}' | base64 -d)
CA_CRT=$(kubectl get secret $SECRET_NAME -n ${NAMESPACE} -o jsonpath='{.data.ca\.crt}' | base64 -d)
APISERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')

cat <<EOF
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: $(echo "$CA_CRT" | base64 | tr -d '\n')
    server: $APISERVER
  name: $CONTEXT-cluster
contexts:
- context:
    cluster: $CONTEXT-cluster
    user: $SA
    namespace: $NAMESPACE
  name: $CONTEXT
current-context: $CONTEXT
users:
- name: $SA
  user:
    token: $TOKEN
EOF