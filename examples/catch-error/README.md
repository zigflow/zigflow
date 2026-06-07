# Catch Error

Catch an error

<!-- toc -->

* [Getting started](#getting-started)
* [Diagram](#diagram)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Getting started

```sh
go run .
```

This will trigger the workflow and print everything to the console.

## Diagram

<!-- ZIGFLOW_GRAPH_START -->
```mermaid
flowchart TD
    catch_error__start([Start])
    catch_error__end([End])
    subgraph try_tryHttp["TRY (tryHttp)"]
        direction TB
        catch_error_tryHttp_try__start([ ])
        catch_error_tryHttp_try__end([ ])
        catch_error_tryHttp_try_http["CALL_HTTP (http)"]
        catch_error_tryHttp_try__start --> catch_error_tryHttp_try_http
        catch_error_tryHttp_try_http --> catch_error_tryHttp_try__end
    end
    subgraph catch_tryHttp["CATCH (tryHttp)"]
        direction TB
        catch_error_tryHttp_catch__start([ ])
        catch_error_tryHttp_catch__end([ ])
        catch_error_tryHttp_catch_dumpEverythingVisibleInCatch["SET (dumpEverythingVisibleInCatch)"]
        catch_error_tryHttp_catch__start --> catch_error_tryHttp_catch_dumpEverythingVisibleInCatch
        catch_error_tryHttp_catch_dumpEverythingVisibleInCatch --> catch_error_tryHttp_catch__end
    end
    catch_error_tryHttp_try__end -.->|"on error"| catch_error_tryHttp_catch__start
    catch_error__start --> catch_error_tryHttp_try__start
    catch_error_tryHttp_try__end --> catch_error__end
```
<!-- ZIGFLOW_GRAPH_END -->
