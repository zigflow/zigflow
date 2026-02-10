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
import CallActivityTask from './call-activity';
import CallGrpcTask from './call-grpc';
import CallHttpTask from './call-http';
import DoTask from './do';
import EmitTask from './emit';
import ForTask from './for';
import ForkTask from './fork';
import ListenTask from './listen';
import RaiseTask from './raise';
import RunTask from './run';
import SetTask from './set';
import SwitchTask from './switch';
import type { Task } from './task';
import TryTask from './try';
import WaitTask from './wait';

export { Task } from './task';

export {
  CallActivityTask,
  CallGrpcTask,
  CallHttpTask,
  DoTask,
  EmitTask,
  ForTask,
  ForkTask,
  ListenTask,
  RaiseTask,
  RunTask,
  SetTask,
  SwitchTask,
  TryTask,
  WaitTask,
};

/**
 * Returns all available Zigflow tasks in alphabetical order by label
 */
export function getTasks(): Task[] {
  return [
    new CallActivityTask(),
    new CallGrpcTask(),
    new CallHttpTask(),
    new DoTask(),
    new EmitTask(),
    new ForTask(),
    new ForkTask(),
    new ListenTask(),
    new RaiseTask(),
    new RunTask(),
    new SetTask(),
    new SwitchTask(),
    new TryTask(),
    new WaitTask(),
  ];
}
