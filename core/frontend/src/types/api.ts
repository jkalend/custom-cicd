export type PipelineStatus = 'pending' | 'running' | 'success' | 'failed' | 'cancelled' | 'never_run';

export interface PipelineStep {
  name: string;
  description: string;
  command: string;
  timeout: number;
  status?: PipelineStatus;
  output?: string;
  error?: string;
}

export interface Pipeline {
  id: string;
  name: string;
  version: string;
  description?: string;
  status: PipelineStatus;
  created_at: string;
  active_runs?: number;
  total_runs?: number;
  last_run_at?: string;
  variables?: Record<string, string>;
  steps: PipelineStep[];
}

export interface PipelineRun {
  id: string;
  pipeline_id: string;
  name: string;
  status: PipelineStatus;
  created_at: string;
  started_at?: string;
  finished_at?: string;
  total_duration?: number;
  steps?: PipelineStep[];
}

export interface CreatePipelineRequest {
  name: string;
  version: string;
  description?: string;
  variables?: Record<string, string>;
  steps: PipelineStep[];
}

export interface ApiResponse<T = any> {
  data?: T;
  error?: string;
}

export interface CreatePipelineResponse {
  pipeline_id: string;
  status: string;
}

export interface CreateAndRunPipelineResponse {
  pipeline_id: string;
  run_id: string;
  status: string;
}

export interface RunPipelineResponse {
  run_id: string;
  status: string;
} 
