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
import { env } from '$env/dynamic/public';
import { readFile, readdir, stat } from 'fs/promises';
import { join, normalize } from 'path';

export interface DirectoryEntry {
  name: string;
  isDirectory: boolean;
  path: string;
}

/**
 * Loads a workflow YAML file from the configured data directory
 *
 * @param workflowPath - Relative path to the workflow file (e.g., 'file.yaml' or 'dir/file.yaml')
 * @returns The workflow file content as a string
 * @throws Error if the file doesn't exist or path traversal is attempted
 */
export async function loadWorkflowFile(workflowPath: string): Promise<string> {
  // Normalize the path to prevent directory traversal attacks
  const normalizedPath = normalize(workflowPath).replace(
    /^(\.\.(\/|\\|$))+/,
    '',
  );

  // Construct the full path
  const fullPath = join(env.PUBLIC_WORKFLOWS_DATA_DIR, normalizedPath);

  // Verify the resolved path is still within the data directory
  if (!fullPath.startsWith(env.PUBLIC_WORKFLOWS_DATA_DIR)) {
    throw new Error('Invalid workflow path: attempted directory traversal');
  }

  try {
    const content = await readFile(fullPath, 'utf-8');
    return content;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === 'ENOENT') {
      throw new Error(`Workflow file not found: ${workflowPath}`);
    }
    throw error;
  }
}

/**
 * Checks if a path is a directory
 *
 * @param workflowPath - Relative path to check
 * @returns true if the path is a directory
 */
export async function isDirectory(workflowPath: string): Promise<boolean> {
  const normalizedPath = normalize(workflowPath).replace(
    /^(\.\.(\/|\\|$))+/,
    '',
  );
  const fullPath = join(env.PUBLIC_WORKFLOWS_DATA_DIR, normalizedPath);

  if (!fullPath.startsWith(env.PUBLIC_WORKFLOWS_DATA_DIR)) {
    throw new Error('Invalid path: attempted directory traversal');
  }

  try {
    const stats = await stat(fullPath);
    return stats.isDirectory();
  } catch {
    return false;
  }
}

/**
 * Lists contents of a directory
 *
 * @param workflowPath - Relative path to the directory (empty string for root)
 * @returns Array of directory entries
 * @throws Error if the path doesn't exist or is not a directory
 */
export async function listDirectory(
  workflowPath: string = '',
): Promise<DirectoryEntry[]> {
  const normalizedPath = normalize(workflowPath || '.').replace(
    /^(\.\.(\/|\\|$))+/,
    '',
  );
  const fullPath = join(env.PUBLIC_WORKFLOWS_DATA_DIR, normalizedPath);

  if (!fullPath.startsWith(env.PUBLIC_WORKFLOWS_DATA_DIR)) {
    throw new Error('Invalid path: attempted directory traversal');
  }

  try {
    const entries = await readdir(fullPath);
    const directoryEntries: DirectoryEntry[] = [];

    for (const entry of entries) {
      const entryPath = join(fullPath, entry);
      const stats = await stat(entryPath);
      const relativePath = workflowPath ? `${workflowPath}/${entry}` : entry;

      directoryEntries.push({
        name: entry,
        isDirectory: stats.isDirectory(),
        path: relativePath,
      });
    }

    // Sort: directories first, then files, alphabetically within each group
    return directoryEntries.sort((a, b) => {
      if (a.isDirectory && !b.isDirectory) return -1;
      if (!a.isDirectory && b.isDirectory) return 1;
      return a.name.localeCompare(b.name);
    });
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === 'ENOENT') {
      throw new Error(`Directory not found: ${workflowPath}`);
    }
    throw error;
  }
}
