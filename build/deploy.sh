set -o errexit
set -o nounset
set -o pipefail

export REGISTRY=quay.io/munnerz/

docker login -e="${QUAY_EMAIL}" -u "${QUAY_USERNAME}" -p "${QUAY_PASSWORD}" quay.io

echo "Pushing images with sha tag."
make push
