#!/bin/bash

for entry in `ls ../logs/createTn*`; do
    $entry
    status=$?
    if test $status -eq 0
    then
	    echo "REMOVE: $entry"
	    rm -f $entry
        status=$?
        if ! test $status -eq 0
        then
            echo "UNABLE to REMOVE $entry"
        fi
    else
	    echo "ERROR: $entry"
    fi
done
