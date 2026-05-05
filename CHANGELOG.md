# Changelog

This changelog is generated from [GitHub Releases](https://github.com/zigflow/zigflow/releases).

## v0.11.1 - 2026-04-20

## What's Changed
* docs: remove h1 title in favour of the logo by @mrsimonemms in https://github.com/zigflow/zigflow/pull/373
* fix: add translation to validation error messages by @mrsimonemms in https://github.com/zigflow/zigflow/pull/376
* feat: add a Zigflow MCP server by @mrsimonemms in https://github.com/zigflow/zigflow/pull/377
* docs: update readme badges by @mrsimonemms in https://github.com/zigflow/zigflow/pull/375


**Full Changelog**: https://github.com/zigflow/zigflow/compare/v0.11.0...v0.11.1

## v0.11.0 - 2026-04-18

## What's Changed
* Add helm smoke tests by @mrsimonemms in https://github.com/zigflow/zigflow/pull/364
* feat(helm): optionally use temporal worker controller deployment by @mrsimonemms in https://github.com/zigflow/zigflow/pull/365
* feat: add ability to use run task with a file by @mrsimonemms in https://github.com/zigflow/zigflow/pull/367
* chore: update dependencies by @mrsimonemms in https://github.com/zigflow/zigflow/pull/369
* chore(deps): bump protobufjs from 7.5.4 to 7.5.5 in /examples/typescript by @dependabot[bot] in https://github.com/zigflow/zigflow/pull/370
* feat: add the zigflow as ascii art in the help text by @mrsimonemms in https://github.com/zigflow/zigflow/pull/371
* Build a core image without python and nodejs installed by @mrsimonemms in https://github.com/zigflow/zigflow/pull/372


**Full Changelog**: https://github.com/zigflow/zigflow/compare/v0.10.2...v0.11.0

## v0.10.2 - 2026-04-12

## What's Changed
* ci: add skaffold version by @mrsimonemms in https://github.com/zigflow/zigflow/pull/352
* feat: add file watcher when running zigflow by @mrsimonemms in https://github.com/zigflow/zigflow/pull/351
* refactor: make the run command more readable by @mrsimonemms in https://github.com/zigflow/zigflow/pull/355
* ci: deploy to homebrew on tag by @mrsimonemms in https://github.com/zigflow/zigflow/pull/357
* Fix some spec mutation issues by @mrsimonemms in https://github.com/zigflow/zigflow/pull/358
* feat: add some additional worker option configuration by @mrsimonemms in https://github.com/zigflow/zigflow/pull/360
* Add priority and fairness demo by @mrsimonemms in https://github.com/zigflow/zigflow/pull/359
* ci: cancel superseded jobs by @mrsimonemms in https://github.com/zigflow/zigflow/pull/361
* feat(run): add --temporal-server-name flag for TLS SNI override by @mrsimonemms in https://github.com/zigflow/zigflow/pull/356


**Full Changelog**: https://github.com/zigflow/zigflow/compare/v0.10.1...v0.10.2

## v0.10.1 - 2026-04-09

## What's Changed
* feat(telemetry): add coarse server country to worker_started event by @mrsimonemms in https://github.com/zigflow/zigflow/pull/346
* feat(telemetry): stabilise anonymous id for Kubernetes deployments by @mrsimonemms in https://github.com/zigflow/zigflow/pull/347
* fix(run): evaluate runtime expressions in map[string]string env values by @mrsimonemms in https://github.com/zigflow/zigflow/pull/348


**Full Changelog**: https://github.com/zigflow/zigflow/compare/v0.10.0...v0.10.1

## v0.10.0 - 2026-04-08

## What's Changed
* docs: add some articles by @mrsimonemms in https://github.com/zigflow/zigflow/pull/317
* docs: improve docs by @mrsimonemms in https://github.com/zigflow/zigflow/pull/318
* feat: enable graceful-shutdown-timer, defaulting to 10s by @mrsimonemms in https://github.com/zigflow/zigflow/pull/320
* ci: add an automated review for each pr by @mrsimonemms in https://github.com/zigflow/zigflow/pull/323
* chore: update dependencies by @mrsimonemms in https://github.com/zigflow/zigflow/pull/324
* fix(helm): always add tmp volume to deployment by @mrsimonemms in https://github.com/zigflow/zigflow/pull/326
* feat: enable multiple workflows by @mrsimonemms in https://github.com/zigflow/zigflow/pull/325
* ci: tighten pr review to give correct file urls by @mrsimonemms in https://github.com/zigflow/zigflow/pull/327
* chore: update dependencies by @mrsimonemms in https://github.com/zigflow/zigflow/pull/328
* chore: update skaffold to run examples by @mrsimonemms in https://github.com/zigflow/zigflow/pull/329
* ci: add script to autoupdate job by @mrsimonemms in https://github.com/zigflow/zigflow/pull/330
* chore: update dependencies by @mrsimonemms in https://github.com/zigflow/zigflow/pull/331
* fix: reduce default start to close timeout to 15s by @mrsimonemms in https://github.com/zigflow/zigflow/pull/321
* Generate the schema by @mrsimonemms in https://github.com/zigflow/zigflow/pull/338
* chore: remove the foreach from the listen task by @mrsimonemms in https://github.com/zigflow/zigflow/pull/343
* [BREAKING]: replace name and namespace with taskQueue and workflowType by @mrsimonemms in https://github.com/zigflow/zigflow/pull/340
* feat: add the zigflow metadata options by @mrsimonemms in https://github.com/zigflow/zigflow/pull/344
* feat: deprecate the top-level timeout by @mrsimonemms in https://github.com/zigflow/zigflow/pull/345


**Full Changelog**: https://github.com/zigflow/zigflow/compare/v0.9.1...v0.10.0

## v0.9.1 - 2026-03-21

## What's Changed
* docs: create testing guide by @mrsimonemms in https://github.com/zigflow/zigflow/pull/307
* chore(deps): bump google.golang.org/grpc from 1.79.2 to 1.79.3 by @dependabot[bot] in https://github.com/zigflow/zigflow/pull/308
* docs: fix typo by @mrsimonemms in https://github.com/zigflow/zigflow/pull/309
* Improve the call HTTP response management by @mrsimonemms in https://github.com/zigflow/zigflow/pull/310
* Fix issues with the for loop task by @mrsimonemms in https://github.com/zigflow/zigflow/pull/311


**Full Changelog**: https://github.com/zigflow/zigflow/compare/v0.9.0...v0.9.1

## v0.9.0 - 2026-03-10

## What's Changed
* [Breaking]: move repo to zigflow org by @mrsimonemms in https://github.com/zigflow/zigflow/pull/298
* ci: set repo owner to helm chart by @mrsimonemms in https://github.com/zigflow/zigflow/pull/299
* feat: harden the docker image and run trivy tests by @mrsimonemms in https://github.com/zigflow/zigflow/pull/300


**Full Changelog**: https://github.com/zigflow/zigflow/compare/v0.8.2...v0.9.0

## v0.8.3-rc2 - 2026-03-09

## What's Changed
* [Breaking]: move repo to zigflow org by @mrsimonemms in https://github.com/zigflow/zigflow/pull/298
* ci: set repo owner to helm chart by @mrsimonemms in https://github.com/zigflow/zigflow/pull/299


**Full Changelog**: https://github.com/zigflow/zigflow/compare/v0.8.2...v0.8.3-rc2

## v0.8.2 - 2026-03-07

## What's Changed
* docs: add contributing instructions by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/254
* chore: configure skaffold dev environment by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/256
* chore(deps): bump ajv from 6.12.6 to 6.14.0 in /docs in the npm_and_yarn group across 1 directory by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/257
* ci: improve helm job speed with caching by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/258
* docs: add star banner to docs by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/262
* docs: add stars to header by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/263
* chore: replace makefile with taskfile by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/264
* chore(e2e): update e2e tasks to use a docker container by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/265
* chore(deps): bump github.com/gofiber/fiber/v2 from 2.52.11 to 2.52.12 in /examples/money-transfer/server in the go_modules group across 1 directory by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/266
* docs: remove additional button by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/267
* Improve docs by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/268
* docs: update readme by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/270
* feat(cmd): create a graph command to generate a mermaid visualisation by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/272
* docs: stylistic updates by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/273
* chore: update dependencies by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/274
* docs: fix broken links by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/277
* chore: update dependencies by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/280
* feat: warn if a non-deterministic fn used outside a set task by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/276
* chore: change to zigflow email by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/281
* fix(docs): hide github nav item on mobile by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/282
* docs: add architectural documentation by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/283
* docs: add explicity mtls config documentation by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/285
* docs: document how to create a dedicated image by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/286
* Remove em dashes by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/287
* docs: fix broken anchors by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/288
* chore(deps): bump the npm_and_yarn group across 1 directory with 2 updates by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/290
* docs: add temporal logo to comparison table by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/291
* feat: improve telemetry with additional richness by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/289
* chore(deps): bump dompurify from 3.3.1 to 3.3.2 in /docs in the npm_and_yarn group across 1 directory by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/292
* docs: fix title colour on dark mode by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/295
* fix golangci-lint errors by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/294
* feat: add update check on run by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/293


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.8.0...v0.8.2

## v0.8.0 - 2026-02-20

## What's Changed
* [Breaking] make commands testable by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/251
* Remote codec server by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/252


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.7.2...v0.8.0

## v0.7.2 - 2026-02-19

## What's Changed
* Sje/release mechanism by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/233
* feat: create a validate command by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/235
* feat(cmd): create a schema command to output the schema by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/236
* docs: add llms.txt by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/237
* refactor: make observability package public by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/238
* Update pre-commit and fix false positives with golangci by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/242
* chore(devcontainer): correct the cloudevent config file by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/241
* fix(tasks): correct field access in exc detail evaluation by @dimostenis in https://github.com/mrsimonemms/zigflow/pull/239
* ci(docs): only run cleanup on non-forks by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/243
* ci(update-dependencies): add go generate by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/245
* chore: monthly dependencies update by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/246
* chore: add security file by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/247


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.7.1...v0.7.2

## v0.7.2-rc5 - 2026-02-15

**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.7.1...v0.7.2-rc5

## v0.7.2-rc4 - 2026-02-15

**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.7.1...v0.7.2-rc4

## v0.7.2-rc3 - 2026-02-15

**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.7.1...v0.7.2-rc3

## v0.7.2-rc2 - 2026-02-15

**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.7.1...v0.7.2-rc2

## v0.7.2-rc1 - 2026-02-15

**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.7.1...v0.7.2-rc1

## v0.7.1 - 2026-02-15

## What's Changed
* chore: add make task to install js dependencies by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/225
* chore: bump go version by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/230
* chore(deps): bump qs from 6.14.1 to 6.14.2 in /docs in the npm_and_yarn group across 1 directory by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/227
* fix(for): add iteration result to state by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/228
* chore: put go version back to v1.25 by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/234


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.7.0...v0.7.1

## v0.7.0 - 2026-02-09

## What's Changed
* ci(stale): never apply stale to anything with 'never-stale' by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/223
* chore: add claude configuration by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/222
* chore(deps): bump github.com/gofiber/fiber/v2 from 2.52.9 to 2.52.11 in /examples/money-transfer/server in the go_modules group across 1 directory by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/224
* feat(cloudevents): implement cloudevents for debugging by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/220


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.6.11...v0.7.0

## v0.6.11 - 2026-01-31

## What's Changed
* chore(deps): bump protobuf from 6.33.0 to 6.33.5 in /examples/python in the uv group across 1 directory by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/213
* fix: clone the arguments without polluting the task by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/216
* chore: monthly dependencies update by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/217


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.6.10...v0.6.11

## v0.6.10 - 2026-01-29

## What's Changed
* Add star button to homepage by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/205
* docs: update --env-prefix docs by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/207
* fix(telemetry): cater for containerised environments by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/208
* fix: switch image's workflow_file envvar to app directory by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/209
* fix: use cmd's context in root command by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/210


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.6.9...v0.6.10

## v0.6.9 - 2026-01-23

## What's Changed
* chore(examples): add encryption example by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/196
* chore: add ga tracking id to docs by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/198
* fix(docs): ignore google tag if envvar not provided by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/201
* ci: ignore uploading docs if external PR by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/202
* chore: monthly dependencies update by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/204
* fix: dont let timer go forever by @dimostenis in https://github.com/mrsimonemms/zigflow/pull/200
* chore(deps): bump the npm_and_yarn group across 2 directories with 1 update by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/203

## New Contributors
* @dimostenis made their first contribution in https://github.com/mrsimonemms/zigflow/pull/200

**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.6.8...v0.6.9

## v0.6.8 - 2026-01-18

## What's Changed
* Add time functions to runtime_expressions by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/195


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.6.7...v0.6.8

## v0.6.7 - 2026-01-15

## What's Changed
* fix: return result from switch tasks by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/193


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.6.6...v0.6.7

## v0.6.6 - 2026-01-11

## What's Changed
* fix: add unit test to check nested tasks are added correctly by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/185
* feat(tasks): stream the response from the run command by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/192


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.6.5...v0.6.6

## v0.6.5 - 2026-01-09

## What's Changed
* fix: prevent nested do tasks being added to the workflow by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/184


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.6.4...v0.6.5

## v0.6.4 - 2026-01-08

## What's Changed
* ci: update pr job to work with checkout v6 by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/178
* chore: monthly dependencies update by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/180
* feat(tasks): configure run container by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/146


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.6.3...v0.6.4

## v0.6.3 - 2026-01-07

## What's Changed
* ci: bump github actions versions by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/175
* Configure a heartbeat by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/177


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.6.2...v0.6.3

## v0.6.2 - 2026-01-06

## What's Changed
* fix: add ambient envvars to run exec command by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/174


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.6.1...v0.6.2

## v0.6.1 - 2026-01-05

## What's Changed
* chore: update copyright year on docs website by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/165
* ci: add preview webpage by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/169
* SEO updates by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/167
* Improve examples by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/170
* chore: tidy up the examples by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/171
* feat: generate cli documentation and publish to docs site by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/172
* fix: change for task repsonse to any by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/173


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.6.0...v0.6.1

## v0.6.0 - 2026-01-02

## What's Changed
* chore: update the tagline to include the word "dsl" by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/160
* Add additional e2e tests by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/103
* chore: monthly dependencies update by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/162
* chore: update the helm chart icon by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/163
* [BREAKING] Update the output and export to SW spec by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/161
* chore: update copyright notice for 2026 by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/164


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.5.0...v0.6.0

## v0.5.0 - 2025-12-25

## What's Changed
* feat: update logo by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/144
* docs: add architectural explanation of temporal by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/145
* refactor: isolate the activities functions in a struct by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/147
* docs: add logo to readme by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/148
* docs: update typo on readme for task queue and workflow type by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/150
* gRPC task by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/135
* docs: add link to logo by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/152
* Update dependencies by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/156
* docs: lint markdown files by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/158
* chore: monthly dependencies update by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/159


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.4.1...v0.5.0

## v0.4.1 - 2025-12-11

## What's Changed
* fix: runtime_expression couldn't handle a slice of strings by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/143


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.4.0...v0.4.1

## v0.4.0 - 2025-12-10

## What's Changed
* chore: add commitlint to makefile by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/138
* Run script by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/140
* [BREAKING] Configure activity options from metadata by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/142


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.3.0...v0.4.0

## v0.3.0 - 2025-12-08

## What's Changed
* feat: implement call activity by @mrsimonemms and @maver1ck  (https://github.com/mrsimonemms/zigflow/pull/136)

**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.2.1...v0.3.0

## v0.2.1 - 2025-12-04

## What's Changed
* fix(listener): add acceptIf on listeners by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/133


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.2.0...v0.2.1

## v0.2.0 - 2025-12-03

## What's Changed
* Update logo by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/114
* chore(deps): bump node-forge from 1.3.1 to 1.3.2 in /docs in the npm_and_yarn group across 1 directory by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/117
* Improve the Helm installation docs by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/116
* feat: add unit tests to the tasks package by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/118
* feat(docs): make the docs homepage prettier by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/120
* ci: automate monthly dependency updates by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/122
* ci: use pat token by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/124
* chore: monthly dependencies update by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/125
* chore: monthly dependencies update by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/127
* feat(tasks): implement continue as new if required by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/126
* Update Helm chart to allow for no workflow requirement by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/129
* chore(deps): bump express from 4.21.2 to 4.22.1 in /docs in the npm_and_yarn group across 1 directory by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/128
* fix(fork): fix flakey test by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/130
* [RFC]: add telemetry to find out zigflow usage by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/119
* Sign the helm chart by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/131


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.1.0...v0.1.1

## What's Changed
* Update logo by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/114
* chore(deps): bump node-forge from 1.3.1 to 1.3.2 in /docs in the npm_and_yarn group across 1 directory by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/117
* Improve the Helm installation docs by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/116
* feat: add unit tests to the tasks package by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/118
* feat(docs): make the docs homepage prettier by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/120
* ci: automate monthly dependency updates by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/122
* ci: use pat token by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/124
* chore: monthly dependencies update by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/125
* chore: monthly dependencies update by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/127
* feat(tasks): implement continue as new if required by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/126
* Update Helm chart to allow for no workflow requirement by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/129
* chore(deps): bump express from 4.21.2 to 4.22.1 in /docs in the npm_and_yarn group across 1 directory by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/128
* fix(fork): fix flakey test by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/130
* [RFC]: add telemetry to find out zigflow usage by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/119
* Sign the helm chart by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/131


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.1.0...v0.2.0

## What's Changed
* Update logo by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/114
* chore(deps): bump node-forge from 1.3.1 to 1.3.2 in /docs in the npm_and_yarn group across 1 directory by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/117
* Improve the Helm installation docs by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/116
* feat: add unit tests to the tasks package by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/118
* feat(docs): make the docs homepage prettier by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/120
* ci: automate monthly dependency updates by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/122
* ci: use pat token by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/124
* chore: monthly dependencies update by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/125
* chore: monthly dependencies update by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/127
* feat(tasks): implement continue as new if required by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/126
* Update Helm chart to allow for no workflow requirement by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/129
* chore(deps): bump express from 4.21.2 to 4.22.1 in /docs in the npm_and_yarn group across 1 directory by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/128
* fix(fork): fix flakey test by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/130
* [RFC]: add telemetry to find out zigflow usage by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/119
* Sign the helm chart by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/131
* chore: monthly dependencies update by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/132


**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.1.0...v0.2.0

## v0.1.0 - 2025-11-24

Complete rewrite from v0.0.7. We're now in public preview 🎉

## What's Changed
* chore: update examples to use connection with envvars by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/52
* Refactor by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/54
* fix(http): handle the call http redirection by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/57
* Configure the input and output by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/63
* chore: add typescript example by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/64
* feat: conditionally execute a task by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/66
* feat: configure multiple workflows by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/67
* Configure query, signal and update listeners by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/68
* feat: configure poller autoscaler by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/69
* feat(tasks): implement flow directive by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/70
* Competing tasks example by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/71
* feat(tasks): implement the switch task by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/72
* feat(task): implement raise error task by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/73
* feat(tasks): run child workflows by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/74
* feat: configure mtls for authentication by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/76
* fix: improve the http args interpolation so it does it correctly by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/78
* Configure search attributes by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/79
* Money transfer demo by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/75
* Demonstrate a change request authorisation workflow by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/83
* fix: remove erroneous print by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/85
* feat(tasks): implement the for task by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/86
* chore: change copyright to Temporal DSL authors by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/87
* fix: check the tasks meet the TaskBuilder interface by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/94
* fix: clone the state for each task by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/96
* feat(tasks): add try catch task by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/92
* docs: update readme for public preview (v0.1.0) by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/91
* Revert "fix: clone the state for each task" by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/101
* fix: load the workflow without validating by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/102
* fix: implement the abstract classes of all the tasks by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/105
* feat: add post load hooks by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/106
* feat: add basic Python example by @maver1ck in https://github.com/mrsimonemms/zigflow/pull/84
* Generate AUTHORS file by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/88
* chore: rename project zigflow by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/107
* chore(deps): bump golang.org/x/crypto from 0.41.0 to 0.45.0 in /examples/money-transfer/server in the go_modules group across 1 directory by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/111
* chore(deps): bump golang.org/x/crypto from 0.43.0 to 0.45.0 in the go_modules group across 1 directory by @dependabot[bot] in https://github.com/mrsimonemms/zigflow/pull/112
* ci: add docs site by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/110
* docs: update docs site by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/113

## New Contributors
* @maver1ck made their first contribution in https://github.com/mrsimonemms/zigflow/pull/84
* @dependabot[bot] made their first contribution in https://github.com/mrsimonemms/zigflow/pull/111

**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.0.7...v0.1.0

## v0.1.0-rc2 - 2025-11-19

## What's Changed
* Revert "fix: clone the state for each task" by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/101
* fix: load the workflow without validating by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/102
* fix: implement the abstract classes of all the tasks by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/105
* feat: add post load hooks by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/106
* feat: add basic Python example by @maver1ck in https://github.com/mrsimonemms/zigflow/pull/84
* Generate AUTHORS file by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/88
* chore: rename project zigflow by @mrsimonemms in https://github.com/mrsimonemms/zigflow/pull/107

## New Contributors
* @maver1ck made their first contribution in https://github.com/mrsimonemms/zigflow/pull/84

**Full Changelog**: https://github.com/mrsimonemms/zigflow/compare/v0.1.0-rc1...v0.1.0-rc2

## v0.1.0-rc1 - 2025-11-08

## What's Changed
* chore: update examples to use connection with envvars by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/52
* Refactor by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/54
* fix(http): handle the call http redirection by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/57
* Configure the input and output by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/63
* chore: add typescript example by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/64
* feat: conditionally execute a task by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/66
* feat: configure multiple workflows by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/67
* Configure query, signal and update listeners by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/68
* feat: configure poller autoscaler by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/69
* feat(tasks): implement flow directive by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/70
* Competing tasks example by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/71
* feat(tasks): implement the switch task by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/72
* feat(task): implement raise error task by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/73
* feat(tasks): run child workflows by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/74
* feat: configure mtls for authentication by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/76
* fix: improve the http args interpolation so it does it correctly by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/78
* Configure search attributes by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/79
* Money transfer demo by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/75
* Demonstrate a change request authorisation workflow by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/83
* fix: remove erroneous print by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/85
* feat(tasks): implement the for task by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/86
* chore: change copyright to Temporal DSL authors by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/87
* fix: check the tasks meet the TaskBuilder interface by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/94
* fix: clone the state for each task by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/96
* feat(tasks): add try catch task by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/92
* docs: update readme for public preview (v0.1.0) by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/91


**Full Changelog**: https://github.com/mrsimonemms/temporal-dsl/compare/v0.0.7...v0.1.0-rc1

## v0.0.7 - 2025-09-08

## What's Changed
* refactor: use a fatal error rather than log.Fatal by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/45
* feat: implement child workflows by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/27
* Fix the competing current tasks by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/48
* ci: move pre-commit to it's own job by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/49
* Add demo to authorise a change request by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/47


**Full Changelog**: https://github.com/mrsimonemms/temporal-dsl/compare/v0.0.6...v0.0.7

## v0.0.6 - 2025-08-29

## What's Changed
* feat: implement schedules by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/26
* Add additional data to the summary for child tasks by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/40
* feat: implement input validation by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/41
* fix: allow examples to connect to cloud or local from envvars by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/43


**Full Changelog**: https://github.com/mrsimonemms/temporal-dsl/compare/v0.0.4...v0.0.6

## v0.0.4 - 2025-08-28

## What's Changed
* A bit of refactoring by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/31
* chore: change docker compose to use temporalio/temporal image by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/32
* feat: add task name as activity context summary by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/33
* feat: add support for switch task by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/17
* chore: add docker compose spec for synapse by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/34
* feat: implement search attributes by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/28
* feat: add metrics and healthcheck handlers by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/4
* fix: add task_key variable for interpolation purposes by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/35
* refactor: tidy up the root init command by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/36
* Don't register the workflow if only contains Do tasks by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/37
* chore: add test for call http activities by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/38
* feat: create a typescript example by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/39


**Full Changelog**: https://github.com/mrsimonemms/temporal-dsl/compare/v0.0.3...v0.0.4

## v0.0.3 - 2025-08-22

## What's Changed
* fix: update state in human in loop part of money transfer by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/20
* refactor: move the if statement checker to a single call by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/21
* chore: rename project as temporal-dsl by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/22
* feat: implement the raise task by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/23
* chore: missed a couple of things to rename by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/24
* docs: add some sexy badges by @mrsimonemms in https://github.com/mrsimonemms/temporal-dsl/pull/25


**Full Changelog**: https://github.com/mrsimonemms/temporal-dsl/compare/v0.0.2...v0.0.3

## v0.0.2 - 2025-08-15

## What's Changed
* feat: add aes encryption by @mrsimonemms in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/5
* feat: improve the interpolation of set values by @mrsimonemms in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/14
* fix: add missing set for default type by @mrsimonemms in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/15
* fix: allow duplicate keys by @mrsimonemms in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/16
* Add a type to the listen task to differentiate updates, queries and signals by @mrsimonemms in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/12
* feat: allow tasks to be conditionally run by @mrsimonemms in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/18
* feat: add signals to listen by @mrsimonemms in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/19
* Do the Money Transfer Demo in TSW by @mrsimonemms in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/10


**Full Changelog**: https://github.com/mrsimonemms/temporal-serverless-workflow/compare/v0.0.1...v0.0.2

## v0.0.1 - 2025-08-13

## What's Changed
* docs: fix typo by @mrsimonemms in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/6
* chore: document how to run examples by @mrsimonemms in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/7
* feat: handle errors on callhttp task by @mrsimonemms in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/8
* Listen task by @mrsimonemms in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/9
* Configure multiple workflows by @mrsimonemms in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/11
* fix: wrap all set values in a side effect to avoid non-determ errors by @mrsimonemms in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/13
* feat: create helm chart by @mrsimonemms in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/1

## New Contributors
* @mrsimonemms made their first contribution in https://github.com/mrsimonemms/temporal-serverless-workflow/pull/6

**Full Changelog**: https://github.com/mrsimonemms/temporal-serverless-workflow/commits/v0.0.1
