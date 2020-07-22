#!/bin/bash
rm -rf repos
mkdir repos 2>/dev/null || exit 1
cd repos || exit 2
#org=LF-Engineering
org=chaoss
git clone "https://github.com/$org/grimoirelab-perceval" || exit 3
git clone "https://github.com/$org/grimoirelab-elk" || exit 4
git clone "https://github.com/$org/grimoirelab-sortinghat" || exit 5
git clone "https://github.com/$org/grimoirelab-kingarthur" || exit 6
#vim --not-a-term -c "%s/pandas==0.18/pandas>=0.18/g" -c 'wq!' grimoirelab-elk/setup.py
#vim --not-a-term -c "%s/redis>=2.10.0, <=2.10.6/redis>=3.0.0/g" -c 'wq!' grimoirelab-elk/setup.py
#vim --not-a-term -c "%s/redis>=2.10.0, <=2.10.6/redis>=3.0.0/g" -c 'wq!' grimoirelab-elk/requirements.txt
#vim --not-a-term -c "%s/redis==3.0.0/redis>=3.0.0/g" -c 'wq!' grimoirelab-kingarthur/setup.py
#vim --not-a-term -c "%s/redis==3.0.0/redis>=3.0.0/g" -c 'wq!' grimoirelab-kingarthur/requirements.txt
#cd ..
#cp patch/utils/p2o.py repos/grimoirelab-elk/utils/
#cp patch/grimoire_elk/elastic_items.py repos/grimoirelab-elk/grimoire_elk/elastic_items.py
#cp patch/grimoire_elk/utils.py repos/grimoirelab-elk/grimoire_elk/utils.py
#vim --not-a-term -c "%s/'group_id': group_id/'group_id': group_id, 'start_msg_num': 0/g" -c 'wq!' grimoirelab-perceval/perceval/backends/core/groupsio.py
#vim --not-a-term -c "%s/if entry is not None:/if entry is not None and 'sortKey' in entry:/g" -c 'wq!' grimoirelab-perceval/perceval/backends/core/gerrit.py
#git diff > api.py.diff
# rocketchat support
cd grimoirelab-elk && git fetch origin pull/906/head:reactions && git checkout reactions && git rebase master && cd ..
#vim --not-a-term -c "%s/if '_id' in usr.keys():/if '_id' in usr.keys() and 'name' in usr.keys():/g" -c 'wq!' grimoirelab-elk/grimoire_elk/enriched/rocketchat.py
# p2o.py per-project affiliations support (specify project via env: PROJECT_SLUG=lfn/onap p2o.py ...)
vim --not-a-term -c "%s/end = Column(DateTime, default=MAX_PERIOD_DATE, nullable=False)\n/end = Column(DateTime, default=MAX_PERIOD_DATE, nullable=False)\r    project_slug = Column(String(128))\r/g" -c 'wq!' grimoirelab-sortinghat/sortinghat/db/model.py || exit 7
cd grimoirelab-sortinghat && git apply ../../patch/api.py.diff && cd .. || exit 8
cd grimoirelab-elk && git apply ../../patch/jira.py.diff && git apply ../../patch/confluence.py.diff && git apply ../../patch/github.py.diff && git apply ../../patch/github2.py.diff && cp ../../patch/identity.py grimoire_elk/enriched/ && cd .. || exit 9
vim --not-a-term -c "%s/PyMySQL==0.9.3/PyMySQL>=0.9.3/g" -c 'wq!' grimoirelab-elk/requirements.txt
vim --not-a-term -c "%s/PyMySQL==0.9.3/PyMySQL>=0.9.3/g" -c 'wq!' grimoirelab-elk/setup.py
#rm -rf repos
