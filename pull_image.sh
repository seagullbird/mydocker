#! /usr/bin/env bash
if [ ! -n "$1" ]; then
	echo "Please input an image name"
	exit 1
fi

docker pull $1
IN=$1
image_name=(${IN//:/ })
docker export $(docker create ${image_name}) > /tmp/${image_name}.tar
rm -rf /var/lib/mydocker/overlay2/${image_name}
mkdir /var/lib/mydocker/overlay2/${image_name}
tar -xf /tmp/${image_name}.tar -C /var/lib/mydocker/overlay2/${image_name}
