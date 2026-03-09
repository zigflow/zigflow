# Copyright 2025 - 2026 Zigflow authors <https://github.com/zigflow/zigflow/graphs/contributors>
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


"""Trigger the Python basic example workflow defined in workflow.yaml."""

from __future__ import annotations

import asyncio
import os
import sys
import uuid
from typing import Any, Dict

from temporalio.client import Client


async def main() -> None:
    address = os.getenv("TEMPORAL_ADDRESS", "localhost:7233")
    namespace = os.getenv("TEMPORAL_NAMESPACE", "default")
    api_key = os.getenv("TEMPORAL_API_KEY")
    tls_enabled = os.getenv("TEMPORAL_TLS", "false").lower() == "true"

    connect_kwargs: Dict[str, Any] = {"namespace": namespace}
    if tls_enabled:
        # Passing True enables TLS using the platform defaults (e.g. system CAs).
        connect_kwargs["tls"] = True
    if api_key:
        connect_kwargs["api_key"] = api_key

    client = await Client.connect(address, **connect_kwargs)

    handle = await client.start_workflow(
        "basic-python",
        {"userId": 3},
        id=f"basic-{uuid.uuid4().hex}",
        task_queue="zigflow",
    )

    print(f"Started workflow: {handle.id}")
    result = await handle.result()
    print(result)


def _run() -> None:
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("Cancelled", file=sys.stderr)
        sys.exit(130)
    except Exception as exc:  # pragma: no cover - minimal CLI
        print(exc, file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    _run()
