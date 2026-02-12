/*
 * Copyright 2025 - 2026 Zigflow authors <https://github.com/mrsimonemms/zigflow/graphs/contributors>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import * as sdk from '@serverlessworkflow/sdk';

import { Task, type TaskState } from './task';

export default class RaiseTask extends Task<
  InstanceType<typeof sdk.Classes.RaiseTask>
> {
  public readonly type = 'raise';
  public readonly label = 'Raise';
  public readonly description = 'Raise an error';

  public getSDKClass(): new (
    data?: TaskState,
  ) => InstanceType<typeof sdk.Classes.RaiseTask> {
    return sdk.Classes.RaiseTask;
  }

  public getDefaultSpecificData(): Record<string, unknown> {
    return {
      raise: {
        error: {
          type: 'https://example.com/errors/my-error',
          title: 'My Error',
          status: 500,
        },
      },
    };
  }
}
