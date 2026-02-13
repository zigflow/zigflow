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
 * Task represents a node in the workflow graph.
 *
 * Each task type extends this base class and provides:
 * - Type identification (type, label, description)
 * - Default configuration (getDefaultData, getDefaultSpecificData)
 * - Schema validation through SDK classes (getSDKClass, validate)
 *
 * The generic parameter T represents the SDK class instance type
 * (e.g., InstanceType<typeof sdk.Classes.SetTask>).
 */

export type TaskState = Record<string, unknown>;

/**
 * Form field types for task configuration UI.
 */
export type FormFieldType =
  | 'text'
  | 'textarea'
  | 'number'
  | 'json'
  | 'duration'
  | 'select';

/**
 * Form field definition for rendering task-specific UI.
 */
export interface FormField {
  /** Unique field identifier */
  id: string;
  /** Field label for display */
  label: string;
  /** Field type */
  type: FormFieldType;
  /** Help text or description */
  helpText?: string;
  /** Placeholder text */
  placeholder?: string;
  /** For select fields: available options */
  options?: Array<{ value: string; label: string }>;
  /** Minimum value for number fields */
  min?: number;
  /** Maximum value for number fields */
  max?: number;
  /** Whether the field is required */
  required?: boolean;
}

/**
 * Base class for all workflow tasks.
 *
 * Responsibilities:
 * - Define task metadata (type, label, description)
 * - Provide default task configuration
 * - Create SDK instances for validation
 * - Define validation behavior
 *
 * Implementation notes:
 * - Tasks are immutable value objects
 * - Task state is serializable JSON
 * - Validation is performed by SDK classes
 * - Each task type maps to an SDK class
 */
export abstract class Task<T = unknown> {
  /**
   * Unique task type identifier (e.g., 'set', 'call-http').
   */
  public abstract readonly type: string;

  /**
   * Human-readable task label (e.g., 'Set', 'Call HTTP').
   */
  public abstract readonly label: string;

  /**
   * Brief task description for UI display.
   */
  public abstract readonly description: string;

  /**
   * Returns the SDK class constructor for this task type.
   *
   * This method is used to:
   * 1. Create SDK instances for validation
   * 2. Determine the correct TypeScript type for task data
   * 3. Enable type-safe task creation and manipulation
   *
   * Example:
   * ```typescript
   * public getSDKClass() {
   *   return sdk.Classes.SetTask;
   * }
   * ```
   *
   * @returns Constructor function for the SDK class
   */
  public abstract getSDKClass(): new (data?: TaskState) => T;

  /**
   * Returns task-specific default configuration.
   *
   * This should return only the fields specific to this task type,
   * not common fields like 'name' or 'if'.
   *
   * Example for SetTask:
   * ```typescript
   * {
   *   set: {
   *     hello: 'world'
   *   }
   * }
   * ```
   *
   * @returns Task-specific configuration object
   */
  public abstract getDefaultSpecificData(): Record<string, unknown>;

  /**
   * Returns form field definitions for this task type.
   *
   * This method defines the UI form structure for configuring
   * task-specific properties. Each task type can provide its own
   * custom form fields based on its schema requirements.
   *
   * @returns Array of form field definitions
   */
  public abstract getFormFields(): FormField[];

  /**
   * Returns complete default task data including optional common fields.
   *
   * This combines optional common fields (metadata, export, output) with
   * task-specific configuration from getDefaultSpecificData().
   *
   * Note: The task name is NOT part of the task state - it's the key in
   * the workflow's 'do' array (e.g., "step1", "wait", "getUser").
   *
   * @returns Complete task configuration object
   */
  public getDefaultData(): TaskState {
    return {
      ...this.getDefaultSpecificData(),
    };
  }

  /**
   * Creates an SDK instance for this task.
   *
   * This is the primary method for instantiating SDK classes.
   * It's used by validate() and can be used by external code
   * that needs access to SDK instances.
   *
   * @param state - Task state to initialize the SDK instance
   * @returns SDK class instance
   */
  public createSDKInstance(state: TaskState): T {
    const SDKClass = this.getSDKClass();
    return new SDKClass(state);
  }

  /**
   * Validates task state using the SDK class.
   *
   * Creates an SDK instance and invokes its validate() method
   * if available. Throws an error if validation fails.
   *
   * Note: This uses a type guard to check for the validate method
   * rather than using 'any' type assertions.
   *
   * @param state - Task state to validate
   * @throws Error if validation fails
   */
  public validate(state: TaskState): void {
    const instance = this.createSDKInstance(state);
    // Type guard to check if instance has a validate method
    if (this.hasValidateMethod(instance)) {
      instance.validate();
    }
  }

  /**
   * Type guard to check if an object has a validate method.
   *
   * @param obj - Object to check
   * @returns True if object has a validate method
   */
  private hasValidateMethod(obj: unknown): obj is { validate: () => void } {
    return (
      typeof obj === 'object' &&
      obj !== null &&
      'validate' in obj &&
      typeof (obj as { validate?: unknown }).validate === 'function'
    );
  }
}
