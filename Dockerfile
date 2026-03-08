# Copyright 2025 - 2026 Zigflow authors <https://github.com/mrsimonemms/zigflow/graphs/contributors>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.26 AS dev
ARG GIT_COMMIT
ARG GIT_REPO="github.com/mrsimonemms/zigflow"
ARG VERSION
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOCACHE=/go/.cache
ENV WORKFLOW_FILE=/go/app/workflow.example.yaml
RUN curl -fsSL https://deb.nodesource.com/setup_lts.x | bash - \
  && apt update \
  && apt install -y nodejs python3 \
  && ln -s /usr/bin/python3 /usr/bin/python \
  && node --version \
  && python --version
USER 1000
WORKDIR /go/app
COPY --chown=1000:1000 go.mod go.sum ./
RUN go mod download
COPY --chown=1000:1000 . .
RUN go generate ./...
COPY --from=cosmtrek/air /go/bin/air /go/bin/air
ENTRYPOINT [ "air" ]

FROM golang:1.26 AS builder
ARG GIT_COMMIT
ARG GIT_REPO="github.com/mrsimonemms/zigflow"
ARG VERSION
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOCACHE=/go/.cache
USER 1000
WORKDIR /go/app
COPY --chown=1000:1000 go.mod go.sum ./
RUN go mod download
COPY --chown=1000:1000 . .
RUN go generate ./... \
  && go build \
  -ldflags \
  "-w -s -X $GIT_REPO/cmd.Version=$VERSION -X $GIT_REPO/cmd.GitCommit=$GIT_COMMIT" \
  -o /go/bin/app

FROM cgr.dev/chainguard/wolfi-base:latest
ARG GIT_COMMIT
ARG VERSION
ENV DISABLE_TELEMETRY=false
ENV GIT_COMMIT="${GIT_COMMIT}"
ENV VERSION="${VERSION}"
ENV WORKFLOW_FILE=/app/workflow.yaml
WORKDIR /app
RUN apk add --no-cache nodejs python3 \
  && node --version \
  && python --version
COPY --from=builder /go/bin/app /app
RUN addgroup -S -g 1000 zigflow && adduser -S -u 1000 zigflow -G zigflow
USER 1000
ENTRYPOINT [ "/app/app" ]
