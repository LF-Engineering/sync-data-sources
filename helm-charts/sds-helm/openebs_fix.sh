#!/bin/bash
prodk.sh delete validatingwebhookconfigurations validation-webhook-cfg
prodk.sh delete deploy openebs-admission-server -n openebs
