#!/bin/bash

echo "####################################################################"
echo "## 2. Namespace: Create"
echo "####################################################################"

source ../init.sh

resp=$(
    curl -H "${AUTH}" -sX POST http://$TumblebugServer/tumblebug/ns -H 'Content-Type: application/json' -d @- <<EOF
        {
			"name": "$NSID",
			"description": "NameSpace for General Testing"
		}
EOF
)
echo ${resp} | jq ''
echo ""

