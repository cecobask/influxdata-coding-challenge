#!/usr/bin/bash

CSV_FILE=${CSV_FILE:-"contacts.csv"}
echo "Importing email requests from '$CSV_FILE'"

MANIFESTS=""
while ((i++)); read -r line
do
  RESOURCE_NAME="emailrequest-$i"
  RECIPIENT_NAME=$(echo "$line" | cut -d ',' -f1)
  RECIPIENT_EMAIL=$(echo "$line" | cut -d ',' -f2)
  export RESOURCE_NAME RECIPIENT_NAME RECIPIENT_EMAIL
  MANIFESTS+="---\n$(envsubst < config/samples/hiring_v1alpha1_emailrequest.yaml)\n"
done < "$CSV_FILE"
echo -e "$MANIFESTS" | kubectl apply -f -
unset i RESOURCE_NAME RECIPIENT_NAME RECIPIENT_EMAIL