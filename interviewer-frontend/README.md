# Legacy interviewer frontend

This source tree is retained temporarily for rollback comparison only. It is no
longer part of the pnpm workspace, Docker/local-dev deployment, or the GitHub
Actions frontend matrix. Interviewer workflows now live in `hr-frontend` under
`/hr/my-interviews`, and port 5175 is assigned to `platform-frontend`.

Do not add new features here. Do not reintroduce `interviewer-frontend` into CI
or `pnpm-workspace.yaml` without an explicit revival TASK. Remove this directory
after the compatibility window closes and production deep links have been
redirected to the staff workspace.
