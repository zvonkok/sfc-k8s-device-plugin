#!/bin/bash

#set -x
set -e

PCI_DEVICES=/sys/bus/pci/devices

cd ${PCI_DEVICES}

VENDOR_ID=0x1924
CLASS_ID=0x020000

for i in `ls ${PCI_DEVICES}`
do
    VENDOR=`cat $i/vendor`
    CLASS=`cat $i/class`

    if [ $(( ${VENDOR} ^ ${VENDOR_ID} )) == 0 ]; then
        if [ $(( ${CLASS} ^ ${CLASS_ID} )) == 0 ]; then
            lshw -c network -businfo | grep $i | awk '{ print $2 }'
        fi

    fi
done
