# Task Type Reference

## set

Set values in `$data`. Only place where `uuid`, `timestamp`,
`timestamp_iso8601` are allowed.

```yaml
- captureInput:
    set:
      requestId: ${ uuid }
      userId: ${ $input.userId }
      greeting: ${ "Hello " + $input.name }
```

Fields merge directly into `$data`. Read as `$data.requestId`, `$data.userId`, etc.

---

## call: http

Make HTTP requests. Result stored in `$data.<taskName>`.

```yaml
- fetchUser:
    call: http
    with:
      method: get
      endpoint: ${ "https://api.example.com/users/" + ($input.id | tostring) }
      headers:
        Authorization: ${ "Bearer " + $env.API_TOKEN }
      body:
        name: ${ $input.name }
```

Properties for `with`:
- `method` (required): get, post, put, patch, delete
- `endpoint` (required): URL string or expression. NOT `url`.
- `headers` (optional): map of header name to value
- `body` (optional): request body (objects become JSON)

Result is an HTTPResponse object with `statusCode`, `headers`, `content`.

---

## call: grpc

Make gRPC calls.

```yaml
- callService:
    call: grpc
    with:
      service: my.service.v1.MyService
      method: GetUser
      proto: https://example.com/protos/service.proto
      arguments:
        user_id: ${ $input.userId }
```

---

## call: activity

Call external Temporal activities written in any SDK language.

```yaml
- fetchProfile:
    call: activity
    with:
      name: activitycall.FetchProfile
      arguments:
        - ${ $data.userId }
        - ${ $data.requestId }
      taskQueue: my-activity-worker
```

---

## do

Group tasks into a named sub-workflow. When at the top level,
each `do` becomes a separate Temporal workflow type.

```yaml
- subFlow:
    do:
      - step1:
          set: { a: 1 }
      - step2:
          call: http
          with:
            method: get
            endpoint: https://example.com
```

---

## for

Loop over arrays, objects or a count.

### Over an array

```yaml
- loopItems:
    for:
      each: item    # variable name for current element
      in: ${ $input.items }
      at: index     # variable name for current index
    do:
      - process:
          set:
            current: ${ $data.item }
            pos: ${ $data.index }
```

### Over an object

```yaml
- loopMap:
    for:
      in: ${ $input.config }
    do:
      - step:
          set:
            key: ${ $data.index }
            value: ${ $data.item }
```

### Repeat N times

```yaml
- repeat5:
    for:
      in: ${ 5 }
    do:
      - step:
          set:
            n: ${ $data.item }
```

### With while condition

```yaml
- loopUntil:
    for:
      in: ${ 10 }
    while: '${ ($output.done // false) == false }'
    do:
      - check:
          output:
            as: ${ { done: ($data.index >= 3) } }
          set:
            iteration: ${ $data.index }
```

---

## fork

Run branches in parallel.

### All branches complete (fan-out)

```yaml
- fanOut:
    fork:
      compete: false
      branches:
        - branchA:
            do:
              - a:
                  set: { result: a }
        - branchB:
            do:
              - b:
                  set: { result: b }
```

### First branch wins (race)

```yaml
- race:
    fork:
      compete: true
      branches:
        - fast:
            do: [...]
        - slow:
            do: [...]
```

---

## switch

Conditional routing. Each case has `when` (expression) and `then` (flow directive).

```yaml
- router:
    switch:
      - electronic:
          when: ${ $input.orderType == "electronic" }
          then: processElectronic
      - physical:
          when: ${ $input.orderType == "physical" }
          then: processPhysical
      - default:
          then: handleUnknown
```

Flow directives for `then`:
- `continue` — proceed to next task in the list
- `exit` — stop this scope, return to parent
- `end` — terminate the entire workflow
- `<taskName>` — redirect to a named task/child workflow

---

## wait

Pause on a Temporal durable timer.

### Duration form

```yaml
- pause:
    wait:
      minutes: 5
      seconds: 30
```

Valid keys: `days`, `hours`, `minutes`, `seconds`, `milliseconds` (all plural).

### Until form (absolute time via expression)

```yaml
- waitUntil:
    wait:
      until: ${ $data.scheduledTime }
```

The `until` value must be an RFC 3339 timestamp string.

---

## listen

Wait for Temporal signals or respond to queries.

### Signal (blocks until received)

```yaml
- awaitApproval:
    listen:
      to:
        one:
          with:
            id: approve       # signal name in Temporal
            type: signal
```

### Query (non-blocking read, responds with data)

```yaml
- queryState:
    listen:
      to:
        one:
          with:
            id: get_state
            type: query
            data:
              status: ${ $data.status }
              progress: ${ $data.progress }
```

### With timeout

```yaml
- timedListen:
    metadata:
      timeout: 30s
    listen:
      to:
        one:
          with:
            id: my-signal
            type: signal
```

---

## try

Error handling with try/catch.

```yaml
- safeCall:
    try:
      - riskyStep:
          call: http
          with:
            method: get
            endpoint: https://might-fail.example.com
    catch:
      do:
        - handleError:
            set:
              error: something went wrong
```

---

## raise

Throw an error to terminate the workflow.

```yaml
- fail:
    raise:
      error:
        type: https://serverlessworkflow.io/spec/1.0.0/errors/communication
        status: 400
```

---

## run

Execute child workflows or containers.

### Child workflow

```yaml
- callChild:
    run:
      workflow:
        type: child-workflow-name
```

### Container (Docker or Kubernetes)

```yaml
- runContainer:
    run:
      container:
        image: alpine:latest
        command: echo "hello"
        environment:
          FOO: bar
```

Container runtime is configured at worker level via `--container-runtime`.
