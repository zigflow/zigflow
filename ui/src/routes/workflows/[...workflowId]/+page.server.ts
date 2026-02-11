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
import { detectTaskTypeFromNode } from '$lib/export/taskTypeDetector';
import {
  isDirectory,
  listDirectory,
  loadWorkflowFile,
} from '$lib/export/workflowLoader';
import {
  type TranslationOptions,
  filterSystemNodes,
  translateGraph,
} from '$lib/export/workflowTranslator';
import { fromYAML } from '$lib/tasks';
import { error } from '@sveltejs/kit';

import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params }) => {
  const workflowPath = params.workflowId;

  // Check if the path is a directory
  const isDir = await isDirectory(workflowPath);

  if (isDir) {
    // Return directory listing
    try {
      const entries = await listDirectory(workflowPath);
      return {
        type: 'directory' as const,
        entries,
        currentPath: workflowPath,
      };
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      throw error(404, `Failed to load directory: ${message}`);
    }
  }

  // It's a file - load the workflow
  let workflowContent: string;
  try {
    workflowContent = await loadWorkflowFile(workflowPath);
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Unknown error';
    throw error(404, `Failed to load workflow: ${message}`);
  }

  const workflow = fromYAML(workflowContent);
  const graph = workflow.toGraph();

  // Configure translation options for Zigflow tasks
  const translationOptions: TranslationOptions = {
    nodeSpacing: { horizontal: 250, vertical: 150 },
    autoLayout: true,
    nodeDataExtractor: (node) => {
      // Extract task-specific data
      const nodeData = node as Record<string, unknown>;

      return {
        label: nodeData.label || nodeData.id || 'Unnamed Node',
        // Preserve any additional metadata
        ...((nodeData.data as Record<string, unknown>) || {}),
      };
    },
  };

  // Translate the graph with Zigflow-specific options
  const translatedGraph = translateGraph(graph, translationOptions);

  // Filter out system entry/exit nodes for cleaner visualization
  const filteredGraph = filterSystemNodes(translatedGraph, {
    keepEntry: false,
    keepExit: false,
  });

  // Detect and apply proper task types to nodes
  const enhancedNodes = filteredGraph.nodes.map((node) => {
    const taskType = detectTaskTypeFromNode(node as Record<string, unknown>);
    return {
      ...node,
      type: taskType,
    };
  });

  return {
    type: 'workflow' as const,
    graph: {
      ...filteredGraph,
      nodes: enhancedNodes,
    },
    workflowYaml: workflowContent,
  };
};
