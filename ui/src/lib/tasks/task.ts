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

/**
 * Common properties that can appear on any task in the workflow
 */
export interface CommonTaskData {
  /** Task name/identifier (the step name in the workflow) */
  name?: string;

  /** Arbitrary metadata for the task */
  metadata?: Record<string, unknown>;

  /** Export data from the task */
  export?: {
    as: Record<string, unknown>;
  };

  /** Output transformation for the task result */
  output?: {
    as: string;
  };
}

/**
 * Complete task data structure combining common and task-specific properties
 */
export interface TaskData {
  /** Common properties shared by all tasks */
  common: CommonTaskData;
  /** Task-specific properties (e.g., for 'set' task: { set: { hello: "world" } }) */
  specific: Record<string, unknown>;
}

/**
 * Abstract base class for Zigflow task types based on the Serverless Workflow specification
 */
export abstract class Task {
  public abstract readonly type: string;
  public abstract readonly label: string;
  public abstract readonly description: string;

  /**
   * Get default task-specific data for this task type.
   * This will be used when creating new nodes in the workflow editor.
   * @returns Default task-specific data structure
   */
  public abstract getDefaultSpecificData(): Record<string, unknown>;

  /**
   * Get default common data for all tasks.
   * Can be overridden by subclasses if needed.
   * @returns Default common data structure
   */
  public getDefaultCommonData(): CommonTaskData {
    return {};
  }

  /**
   * Get complete default task data combining common and specific properties.
   * @returns Complete default task data
   */
  public getDefaultTaskData(): TaskData {
    return {
      common: this.getDefaultCommonData(),
      specific: this.getDefaultSpecificData(),
    };
  }
}
