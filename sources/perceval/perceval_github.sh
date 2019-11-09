#!/bin/bash
if [ -z "${API_KEY}" ]
then
  echo -n "GitHub API key: "
  read -s API_KEY
fi
perceval github opennetworkinglab onos --category issue -t "${API_KEY}" --sleep-for-rate > .perceval_onos.github.log || exit 2
perceval github opennetworkinglab onos --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
perceval github opennetworkinglab spring-open --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
perceval github opennetworkinglab spring-open --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
perceval github opennetworkinglab OnosSystemTest --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
perceval github opennetworkinglab OnosSystemTest --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
perceval github opennetworkinglab onos-loxi --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
perceval github opennetworkinglab onos-loxi --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
perceval github opennetworkinglab onos-yang-tools --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
perceval github opennetworkinglab onos-yang-tools --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
perceval github opennetworkinglab onos-app-samples --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
perceval github opennetworkinglab onos-app-samples --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
perceval github opennetworkinglab routing --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
perceval github opennetworkinglab routing --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
perceval github opennetworkinglab spring-open-cli --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
perceval github opennetworkinglab spring-open-cli --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab onos-felix --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab onos-felix --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab fabric-control --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab fabric-control --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab onos-vm --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab onos-vm --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab vm-build --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab vm-build --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab olt-oftest --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab olt-oftest --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab onos-nemo --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab onos-nemo --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab onos-gerrit-base --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab onos-gerrit-base --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab manifest --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab manifest --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab OnosSystemTestJenkins --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab OnosSystemTestJenkins --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab cord-openwrt --category issue -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
#perceval github opennetworkinglab cord-openwrt --category pull_request -t "${API_KEY}" --sleep-for-rate >> .perceval_onos.github.log || exit 2
