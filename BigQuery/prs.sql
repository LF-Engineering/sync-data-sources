select
  JSON_EXTRACT(payload, '$.pull_request.number') as id
from (
  select
    *
  from
    TABLE_DATE_RANGE([githubarchive:day.],TIMESTAMP('{{dtfrom}}'),TIMESTAMP('{{dtto}}'))
  )
where
  org.login = '{{org}}'
  and repo.name = '{{repo}}'
group by
  id
limit
  1000000
;
