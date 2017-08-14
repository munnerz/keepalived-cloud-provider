set -o errexit
set -o nounset

export REGISTRY=quay.io/munnerz/

docker login -e="${QUAY_EMAIL}" -u "${QUAY_USERNAME}" -p "${QUAY_PASSWORD}" quay.io

if [ "${TRAVIS_TAG}" = "" ]; then
    echo "Pushing images with sha tag."
    make push
else
    echo "Pushing images with release tag."
    make push MUTABLE_TAG=latest VERSION="${TRAVIS_TAG}"
fi
